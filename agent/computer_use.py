"""
Claude Computer Use loop for browser automation.
Takes screenshots, sends them to Claude, executes returned actions.
"""

import asyncio
import base64
import json
from typing import AsyncGenerator, Callable, Optional

import anthropic
from playwright.async_api import Page

ANTHROPIC_API_KEY = __import__("os").environ.get("ANTHROPIC_API_KEY", "")
MODEL = "claude-opus-4-5"
MAX_ITERATIONS = 50
SCREENSHOT_WIDTH = 1280
SCREENSHOT_HEIGHT = 800


def _build_system_prompt(job) -> str:
    has_creds = bool(job.credential_username)
    has_card = bool(job.card_number)
    cred_part = (
        f"Username/email: {job.credential_username}\nPassword: {job.credential_password}\n"
        if has_creds else "No credentials provided — look for guest payment options.\n"
    )
    card_part = (
        f"Card number: {job.card_number}\nExpiry: {job.card_expiry}\nCVV: {job.card_cvv}\nHolder: {job.card_holder}\n"
        if has_card else "No card provided — use RF/IRIS code payment if available.\n"
    )
    rf_part = f"RF/payment code: {job.payment_code}\n" if job.payment_code else ""

    return f"""You are an AI payment agent for the BillTracker app. Your task is to pay the following bill:

Provider: {job.provider_name}
Amount: {job.amount:.2f} EUR
{rf_part}
Credentials:
{cred_part}
Card details:
{card_part}

INSTRUCTIONS:
1. Navigate to the correct payment page for this provider.
2. Log in if needed using the provided credentials.
3. Find the bill payment section. Look for the specific amount {job.amount:.2f}€ or the RF code.
4. If a login OTP/verification code is required via SMS, say exactly: "OTP_REQUIRED" and wait.
5. Enter card details when prompted.
6. If a payment confirmation OTP is required via SMS, say exactly: "OTP_REQUIRED" and wait.
7. Complete the payment and confirm success.
8. When done, say exactly: "PAYMENT_COMPLETE" followed by any confirmation reference number.

IMPORTANT RULES:
- Only pay the exact amount: {job.amount:.2f} EUR
- Do NOT navigate away from payment flows
- Do NOT accept any extra charges or subscriptions
- If you see a CAPTCHA you cannot solve, say: "CAPTCHA_BLOCKED"
- If the amount shown does not match {job.amount:.2f}€ exactly, say: "AMOUNT_MISMATCH: [amount shown]"
- Communicate progress in Greek (e.g. "Συνδέομαι...", "Πληρώνω...", "Επιβεβαίωση...")
- If you are unsure about an action, describe what you see before proceeding
"""


async def run_computer_use_loop(
    page: Page,
    system_prompt: str,
    job,
    otp_queue: asyncio.Queue,
    emit: Callable,
) -> AsyncGenerator[dict, None]:
    """Main Computer Use loop."""
    client = anthropic.AsyncAnthropic(api_key=ANTHROPIC_API_KEY)

    messages: list[dict] = []
    iteration = 0

    # Initial screenshot to start the loop
    screenshot_b64 = await _take_screenshot(page)
    messages.append({
        "role": "user",
        "content": [
            {
                "type": "tool_result",
                "tool_use_id": "initial",
                "content": [{"type": "image", "source": {"type": "base64", "media_type": "image/png", "data": screenshot_b64}}],
            }
        ],
    })
    # First turn: tell Claude to start
    messages = [{"role": "user", "content": f"Ξεκίνα την πληρωμή. Τρέχουσα σελίδα:\n[screenshot attached]"}]
    messages[0]["content"] = [
        {"type": "text", "text": "Ξεκίνα την πληρωμή. Αυτή είναι η τρέχουσα σελίδα:"},
        {"type": "image", "source": {"type": "base64", "media_type": "image/png", "data": screenshot_b64}},
    ]

    tools = [
        {
            "type": "computer_20241022",
            "name": "computer",
            "display_width_px": SCREENSHOT_WIDTH,
            "display_height_px": SCREENSHOT_HEIGHT,
            "display_number": 1,
        }
    ]

    while iteration < MAX_ITERATIONS:
        iteration += 1

        response = await client.beta.messages.create(
            model=MODEL,
            max_tokens=4096,
            system=system_prompt,
            messages=messages,
            tools=tools,
            betas=["computer-use-2024-10-22"],
        )

        # Extract text messages from Claude
        text_parts = [b.text for b in response.content if hasattr(b, "text") and b.text]
        for text in text_parts:
            yield emit("agent_message", message=text)

            if "PAYMENT_COMPLETE" in text:
                ref = text.split("PAYMENT_COMPLETE")[-1].strip()
                yield emit("success", message="Η πληρωμή ολοκληρώθηκε! ✓", reference=ref)
                return

            if "CAPTCHA_BLOCKED" in text:
                yield emit("error", message="Αδύνατη η αυτόματη πληρωμή: CAPTCHA εντοπίστηκε.")
                return

            if "AMOUNT_MISMATCH" in text:
                yield emit("error", message=f"Διαφορά ποσού: {text}")
                return

        # Check for tool use (computer actions)
        tool_use_blocks = [b for b in response.content if b.type == "tool_use"]

        if not tool_use_blocks:
            # Claude is done thinking, no more actions
            if response.stop_reason == "end_turn":
                yield emit("info", message="Ο agent ολοκλήρωσε χωρίς επιβεβαίωση.")
            break

        # Add assistant response to messages
        messages.append({"role": "assistant", "content": response.content})

        # Execute each tool action
        tool_results = []
        for tool_use in tool_use_blocks:
            if tool_use.name != "computer":
                continue

            action = tool_use.input.get("action")
            yield emit("action", action=action, details=_describe_action(tool_use.input))

            # Handle OTP_REQUIRED before executing actions
            # (Claude will type OTP_REQUIRED as text before the action)
            if any("OTP_REQUIRED" in t for t in text_parts):
                yield emit("otp_required", message="Εισάγετε τον κωδικό OTP που λάβατε στο κινητό σας")
                try:
                    otp_code = await asyncio.wait_for(otp_queue.get(), timeout=120.0)
                    yield emit("otp_received", message="OTP λήφθηκε, συνεχίζω...")
                    # Type the OTP
                    await page.keyboard.type(otp_code)
                    await page.keyboard.press("Enter")
                    await asyncio.sleep(2)
                except asyncio.TimeoutError:
                    yield emit("error", message="Timeout: δεν λήφθηκε OTP εντός 2 λεπτών.")
                    return

            result_content = await _execute_action(page, tool_use.input)
            tool_results.append({
                "type": "tool_result",
                "tool_use_id": tool_use.id,
                "content": result_content,
            })

        # Add tool results to messages
        messages.append({"role": "user", "content": tool_results})

    if iteration >= MAX_ITERATIONS:
        yield emit("error", message="Ο agent έφτασε τον μέγιστο αριθμό ενεργειών.")


async def _take_screenshot(page: Page) -> str:
    """Take a screenshot and return as base64."""
    screenshot_bytes = await page.screenshot(type="png", full_page=False)
    return base64.b64encode(screenshot_bytes).decode()


async def _execute_action(page: Page, action_input: dict) -> list[dict]:
    """Execute a computer use action and return screenshot result."""
    action = action_input.get("action")

    try:
        if action == "screenshot":
            pass  # just take screenshot below

        elif action == "left_click":
            x, y = action_input["coordinate"]
            await page.mouse.click(x, y)
            await asyncio.sleep(0.5)

        elif action == "double_click":
            x, y = action_input["coordinate"]
            await page.mouse.dblclick(x, y)
            await asyncio.sleep(0.5)

        elif action == "right_click":
            x, y = action_input["coordinate"]
            await page.mouse.click(x, y, button="right")
            await asyncio.sleep(0.5)

        elif action == "type":
            text = action_input.get("text", "")
            await page.keyboard.type(text, delay=50)
            await asyncio.sleep(0.3)

        elif action == "key":
            key = action_input.get("key", "")
            await page.keyboard.press(key)
            await asyncio.sleep(0.5)

        elif action == "scroll":
            x, y = action_input["coordinate"]
            direction = action_input.get("direction", "down")
            amount = action_input.get("amount", 3)
            delta = amount * 100 * (1 if direction == "down" else -1)
            await page.mouse.wheel(0, delta)
            await asyncio.sleep(0.3)

        elif action == "mouse_move":
            x, y = action_input["coordinate"]
            await page.mouse.move(x, y)

        elif action == "left_click_drag":
            start = action_input["start_coordinate"]
            end = action_input["coordinate"]
            await page.mouse.move(start[0], start[1])
            await page.mouse.down()
            await page.mouse.move(end[0], end[1])
            await page.mouse.up()

        # Wait for any navigation/animations
        await page.wait_for_load_state("domcontentloaded", timeout=5000)

    except Exception:
        pass  # don't fail on action errors, just take screenshot

    # Always return a screenshot
    screenshot_b64 = await _take_screenshot(page)
    return [
        {
            "type": "image",
            "source": {"type": "base64", "media_type": "image/png", "data": screenshot_b64},
        }
    ]


def _describe_action(action_input: dict) -> str:
    action = action_input.get("action", "")
    if action == "left_click":
        coord = action_input.get("coordinate", [])
        return f"Click at {coord}"
    elif action == "type":
        text = action_input.get("text", "")
        # mask card/password-like strings
        if len(text) >= 13 and text.replace(" ", "").isdigit():
            return f"Type card number (masked)"
        return f"Type: {text[:30]}{'...' if len(text) > 30 else ''}"
    elif action == "key":
        return f"Press {action_input.get('key', '')}"
    elif action == "scroll":
        return f"Scroll {action_input.get('direction', '')} {action_input.get('amount', '')}x"
    elif action == "screenshot":
        return "Screenshot"
    return action
