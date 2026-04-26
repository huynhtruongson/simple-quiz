import type { CreateRoomRequest, ListQuizzesResponse, Room } from './types'

const JSON_HEADERS = { 'Content-Type': 'application/json' }

async function parseJson<T>(response: Response): Promise<T> {
  if (!response.ok) {
    const fallbackMessage = `Request failed with status ${response.status}`
    try {
      const data = (await response.json()) as { error?: string }
      if (typeof data.error === 'string' && data.error.trim().length > 0) {
        throw new Error(data.error)
      }
      throw new Error(fallbackMessage)
    } catch (error) {
      if (error instanceof Error && error.message !== fallbackMessage) {
        throw error
      }
      throw new Error(fallbackMessage, { cause: error })
    }
  }
  return (await response.json()) as T
}

export async function fetchQuizzes(): Promise<ListQuizzesResponse> {
  const response = await fetch('/quizzes')
  return parseJson<ListQuizzesResponse>(response)
}

export async function createRoom(payload: CreateRoomRequest): Promise<Room> {
  const response = await fetch('/rooms', {
    method: 'POST',
    headers: JSON_HEADERS,
    body: JSON.stringify(payload),
  })
  return parseJson<Room>(response)
}

export async function validateRoom(roomID: string): Promise<void> {
  const response = await fetch(`/rooms/${encodeURIComponent(roomID)}`)
  await parseJson(response)
}
