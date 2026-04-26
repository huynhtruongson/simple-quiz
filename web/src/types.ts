export type SessionState = 'waiting' | 'question_active' | 'question_result' | 'finished'

export interface QuizQuestion {
  id: string
  text: string
  options: string[]
  answer: number
}

export interface Quiz {
  id: string
  title: string
  total_question: number
}

export interface ListQuizzesResponse {
  quizzes: Quiz[]
}

export interface CreateRoomRequest {
  quiz_id: string
}

export interface Room {
  room_id: string
  quiz_id: string
}

export interface Player {
  id: string
  name: string
  is_host: boolean
  score: number
}

export interface LeaderboardEntry {
  player_id:string
  rank: number
  name: string
  score: number
}

export interface IncomingWsMessage {
  type: 'join' | 'start_quiz' | 'submit_answer'
  quiz_id?: string
  name?: string
  question_id?: string
  answer_index?: number
}

export interface SessionJoinedPayload {
  player_id: string
  name: string
  is_host: boolean
  player_list: Player[]
  leaderboard: LeaderboardEntry[]
}

export interface PlayerJoinedPayload {
  player_id: string
  name: string
  player_list: Player[]
  leaderboard: LeaderboardEntry[]
}

export interface PlayerLeftPayload {
  host_player_id: string
  player_list: Player[]
  leaderboard: LeaderboardEntry[]
}

export interface QuizStartedPayload {
  total_questions: number
}

export interface QuestionShowPayload {
  question_id: string
  text: string
  options: string[]
  index: number
  total: number
  ends_at: number
}

export interface AnswerAcceptedPayload {
  question_id: string
  points_earned: number
  total_score: number
  correct: boolean
}

export interface QuestionResultPayload {
  correct_answer: number
  correct_text: string
  player_results: Record<string, number>
  leaderboard: LeaderboardEntry[]
}

export interface QuizFinishedPayload {
  final_leaderboard: LeaderboardEntry[]
}

export interface ErrorPayload {
  code: string
  message: string
}

export interface OutgoingWsMessage<T = unknown> {
  type: string
  payload?: T
}
