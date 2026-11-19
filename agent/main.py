"""
BillTracker AI Payment Agent
FastAPI service that orchestrates browser automation + Claude Computer Use
to automatically pay bills on behalf of the user.
"""

import asyncio
import base64
import json
import os
import uuid
from typing import AsyncGenerator, Optional

import anthropic
from fastapi import FastAPI, HTTPException, Header
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import StreamingResponse
from pydantic import BaseModel
from playwright.async_api import async_playwright, Browser, BrowserContext, Page

from computer_use import run_computer_use_loop

app = FastAPI(title="BillTracker Agent", version="1.0.0")

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)

# Active sessions: job_id → asyncio.Queue for OTP input
otp_queues: dict[str, asyncio.Queue] = {}

AGENT_SECRET = os.environ.get("AGENT_SECRET", "dev-secret")


class PaymentJob(BaseModel):
    job_id: str
    bill_id: str
    provider_name: str
    amount: float
    payment_code: Optional[str] = None
    payment_url: Optional[str] = None
    credential_label: Optional[str] = None
    credential_username: Optional[str] = None
    credential_password: Optional[str] = None
    card_number: Optional[str] = None
    card_expiry: Optional[str] = None
    card_cvv: Optional[str] = None
    card_holder: Optional[str] = None


class OTPInput(BaseModel):
    otp: str


def verify_secret(x_agent_secret: str = Header(...)):
    if x_agent_secret != AGENT_SECRET:
        raise HTTPException(status_code=401, detail="Unauthorized")


@app.post("/pay")
async def start_payment(job: PaymentJob, x_agent_secret: str = Header(...)):
    """Start a payment job and stream SSE events back."""
    verify_secret(x_agent_secret)

    otp_queue: asyncio.Queue = asyncio.Queue()
    otp_queues[job.job_id] = otp_queue

    async def event_stream() -> AsyncGenerator[str, None]:
        try:
            async for event in run_payment_agent(job, otp_queue):
                yield f"data: {json.dumps(event)}\n\n"
        except Exception as e:
            yield f"data: {json.dumps({'type': 'error', 'message': str(e)})}\n\n"
        finally:
            otp_queues.pop(job.job_id, None)
            yield "data: {\"type\": \"done\"}\n\n"

    return StreamingResponse(
        event_stream(),
        media_type="text/event-stream",
        headers={
            "Cache-Control": "no-cache",
            "X-Accel-Buffering": "no",
        },
    )


@app.post("/otp/{job_id}")
async def submit_otp(job_id: str, body: OTPInput, x_agent_secret: str = Header(...)):
    """Submit OTP for a waiting payment job."""
    verify_secret(x_agent_secret)
    queue = otp_queues.get(job_id)
    if not queue:
        raise HTTPException(status_code=404, detail="Job not found or already completed")
    await queue.put(body.otp)
    return {"ok": True}


@app.get("/health")
async def health():
    return {"status": "ok"}


async def run_payment_agent(job: PaymentJob, otp_queue: asyncio.Queue) -> AsyncGenerator[dict, None]:
    """Main agent loop: browser + Claude Computer Use."""

    def emit(type: str, **kwargs) -> dict:
        return {"type": type, "job_id": job.job_id, **kwargs}

    yield emit("started", message=f"Ξεκινώ πληρωμή {job.provider_name} — {job.amount:.2f}€")

    async with async_playwright() as pw:
        browser: Browser = await pw.chromium.launch(
            headless=True,
            args=["--no-sandbox", "--disable-dev-shm-usage"],
        )
        context: BrowserContext = await browser.new_context(
            viewport={"width": 1280, "height": 800},
            locale="el-GR",
            timezone_id="Europe/Athens",
        )
        page: Page = await context.new_page()

        try:
            start_url = job.payment_url or _infer_url(job.provider_name)
            yield emit("navigating", message=f"Ανοίγω {start_url}")
            await page.goto(start_url, wait_until="domcontentloaded", timeout=30000)

            system_prompt = _build_system_prompt(job)

            async for event in run_computer_use_loop(
                page=page,
                system_prompt=system_prompt,
                job=job,
                otp_queue=otp_queue,
                emit=emit,
            ):
                yield event

        except Exception as e:
            yield emit("error", message=f"Σφάλμα: {str(e)}")
        finally:
            await browser.close()


def _infer_url(provider_name: str) -> str:
    """Map provider name to login URL."""
    name = provider_name.lower()
    urls = {
        "taxisnet": "https://www.aade.gr/myaade",
        "aade": "https://www.aade.gr/myaade",
        "εφορία": "https://www.aade.gr/myaade",
        "deh": "https://www.dei.gr/el/idiotikes-katikies/logariasmos-kai-pliromi/ilektroniki-pliromi/",
        "δεη": "https://www.dei.gr/el/idiotikes-katikies/logariasmos-kai-pliromi/ilektroniki-pliromi/",
        "vodafone": "https://www.vodafone.gr/my-vodafone/",
        "cosmote": "https://www.cosmote.gr/cs/mycosmote/",
        "nova": "https://www.nova.gr/my-nova/",
        "wind": "https://www.wind.gr/my-wind/",
        "eydap": "https://www.eydap.gr/CustomerService/ElectronicServices/",
        "ευδαπ": "https://www.eydap.gr/CustomerService/ElectronicServices/",
        "elta": "https://www.elta.gr/",
    }
    for key, url in urls.items():
        if key in name:
            return url
    # fallback: google search for the provider
    return f"https://www.google.com/search?q={provider_name}+πληρωμή+λογαριασμού+online"
