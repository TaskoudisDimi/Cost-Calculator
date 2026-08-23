import { ref } from 'vue'
import { ref as storageRef, uploadBytes, getDownloadURL, deleteObject } from 'firebase/storage'
import { storage, auth } from '@/firebase'

export function useReceiptUpload() {
  const uploading = ref(false)
  const error = ref('')

  async function upload(billId: string, file: File): Promise<string> {
    const uid = auth.currentUser?.uid
    if (!uid) throw new Error('Not authenticated')

    const ext = file.name.split('.').pop()?.toLowerCase() ?? 'bin'
    const path = `receipts/${uid}/${billId}/${Date.now()}.${ext}`
    const fileRef = storageRef(storage, path)

    uploading.value = true
    error.value = ''
    try {
      await uploadBytes(fileRef, file, { contentType: file.type })
      return await getDownloadURL(fileRef)
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'Upload failed'
      throw e
    } finally {
      uploading.value = false
    }
  }

  async function remove(url: string): Promise<void> {
    try {
      const fileRef = storageRef(storage, url)
      await deleteObject(fileRef)
    } catch {
      // ignore — file may already be gone
    }
  }

  return { uploading, error, upload, remove }
}
