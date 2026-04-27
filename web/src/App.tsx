import { useEffect, useMemo, useRef, useState } from 'react'
import type { FormEvent } from 'react'
import { createRoom, fetchQuizzes, validateRoom } from './api'
import './styles.css'
import type {
  AnswerAcceptedPayload,
  ErrorPayload,
  LeaderboardEntry,
  OutgoingWsMessage,
  Player,
  QuestionResultPayload,
  QuestionShowPayload,
  Quiz,
  QuizFinishedPayload,
  SessionJoinedPayload,
  PlayerJoinedPayload,
  PlayerLeftPayload,
  SessionState,
} from './types'
import { QuizWsClient } from './ws'

type View = 'lobby' | 'room'

const ANSWER_LABELS = ['A', 'B', 'C', 'D']

function App() {
  const wsClientRef = useRef<QuizWsClient | null>(null)
  const playerIDRef = useRef<string | null>(null)

  const [quizzes, setQuizzes] = useState<Quiz[]>([])
  const [loadingQuizzes, setLoadingQuizzes] = useState(true)
  const [lobbyError, setLobbyError] = useState('')
  const [roomError, setRoomError] = useState('')
  const [connectionStatus, setConnectionStatus] = useState('Disconnected')
  const [view, setView] = useState<View>('lobby')

  const [displayName, setDisplayName] = useState('')
  const [quickJoinRoomID, setQuickJoinRoomID] = useState('')

  const [roomID, setRoomID] = useState('')
  const [players, setPlayers] = useState<Player[]>([])
  const [isHost, setIsHost] = useState(false)
  const [sessionState, setSessionState] = useState<SessionState>('waiting')
  const [totalQuestions, setTotalQuestions] = useState(0)
  const [question, setQuestion] = useState<QuestionShowPayload | null>(null)
  const [questionResult, setQuestionResult] = useState<QuestionResultPayload | null>(null)
  const [leaderBoard, setLeaderboard] = useState<LeaderboardEntry[] | null>(null)
  const [selectedAnswerIndex, setSelectedAnswerIndex] = useState<number | null>(null)
  const [answerAccepted, setAnswerAccepted] = useState<AnswerAcceptedPayload | null>(null)
  const [nowMs, setNowMs] = useState(() => Date.now())
  
  useEffect(() => {
    let cancelled = false

    fetchQuizzes()
      .then((response) => {
        if (cancelled) {
          return
        }
        setQuizzes(response.quizzes)
      })
      .catch((error: unknown) => {
        if (cancelled) {
          return
        }
        const message = error instanceof Error ? error.message : 'Failed to load quizzes'
        setLobbyError(message)
      })
      .finally(() => {
        if (!cancelled) {
          setLoadingQuizzes(false)
        }
      })

    return () => {
      cancelled = true
    }
  }, [])

    useEffect(() => {
    if (!question || sessionState !== "question_active") return;

    const tick = () => setNowMs(Date.now());

    tick(); 

    const timer = window.setInterval(tick, 1000);

    return () => window.clearInterval(timer);
  }, [question?.ends_at, sessionState]);

  useEffect(() => {
    return () => {
      wsClientRef.current?.close()
    }
  }, [])

  const canSubmitAnswer = useMemo(
    () => sessionState === 'question_active' && selectedAnswerIndex !== null && !answerAccepted,
    [sessionState, selectedAnswerIndex, answerAccepted],
  )

   const secondsLeft =
    !question || sessionState !== "question_active"
      ? 0
      : Math.max(
          0,
          Math.ceil((question.ends_at * 1000 - nowMs) / 1000)
        );

  const connectToRoom = (targetRoomID: string, quizID: string, playerName: string) => {
    const sanitizedRoomID = targetRoomID.trim()
    const sanitizedName = playerName.trim()

    if (!sanitizedRoomID) {
      setLobbyError('Room ID is required')
      return
    }
    if (!sanitizedName) {
      setLobbyError('Display name is required')
      return
    }

    setLobbyError('')
    setRoomError('')
    setConnectionStatus('Connecting...')
    setView('room')
    setRoomID(sanitizedRoomID)
    setPlayers([])
    setQuestion(null)
    setQuestionResult(null)
    setLeaderboard(null)
    setSelectedAnswerIndex(null)
    setAnswerAccepted(null)
    setSessionState('waiting')
    setTotalQuestions(0)

    wsClientRef.current?.close()
    const client = new QuizWsClient(sanitizedRoomID, {
      onMessage: (message) => handleServerMessage(message),
      onClose: () => {
        setConnectionStatus('Disconnected')
      },
      onError: () => {
        setRoomError('WebSocket connection failed')
      },
    })
    wsClientRef.current = client

    client.onOpen(() => {
      setConnectionStatus('Connected')
      client.send({
        type: 'join',
        quiz_id: quizID,
        name: sanitizedName,
      })
    })
  }

  const resetToLobby = () => {
    wsClientRef.current?.close()
    wsClientRef.current = null
    playerIDRef.current = null
    setView('lobby')
    setConnectionStatus('Disconnected')
    setRoomID('')
    setPlayers([])
    setIsHost(false)
    setSessionState('waiting')
    setQuestion(null)
    setQuestionResult(null)
    setLeaderboard(null)
    setSelectedAnswerIndex(null)
    setAnswerAccepted(null)
    setRoomError('')
  }

  const handleServerMessage = (message: OutgoingWsMessage) => {
    switch (message.type) {
      case 'session_joined': {
        const payload = message.payload as SessionJoinedPayload
        playerIDRef.current = payload.player_id
        setIsHost(payload.is_host)
        setPlayers(payload.player_list)
        setRoomError('')
        break
      }
      case 'player_joined': {
        const payload = message.payload as PlayerJoinedPayload
        setPlayers(payload.player_list)
        setLeaderboard(payload.leaderboard)
        break
      }
      case 'player_left': {
        const payload = message.payload as PlayerLeftPayload
        setIsHost(payload.host_player_id === playerIDRef.current)
        setPlayers(payload.player_list)
        setLeaderboard(payload.leaderboard)
        break
      }
      case 'quiz_started': {
        const payload = message.payload as { total_questions: number }
        setTotalQuestions(payload.total_questions)
        setSessionState('question_active')
        setQuestionResult(null)
        setAnswerAccepted(null)
        break
      }
      case 'question_show': {
        const payload = message.payload as QuestionShowPayload
        setQuestion(payload)
        setSessionState('question_active')
        setSelectedAnswerIndex(null)
        setAnswerAccepted(null)
        setQuestionResult(null)
        break
      }
      case 'answer_accepted': {
        const payload = message.payload as AnswerAcceptedPayload
        setAnswerAccepted(payload)
        break
      }
      case 'question_result': {
        const payload = message.payload as QuestionResultPayload
        setQuestionResult(payload)
        setLeaderboard(payload.leaderboard)
        setSessionState('question_result')
        break
      }
      case 'quiz_finished': {
        const payload = message.payload as QuizFinishedPayload
        setLeaderboard(payload.final_leaderboard)
        setSessionState('finished')
        break
      }
      case 'error': {
        const payload = message.payload as ErrorPayload
        setRoomError(payload.message)
        break
      }
      default:
        break
    }
  }

  const handleCreateRoomForQuiz = async (quizID: string) => {
    if (!displayName.trim()) {
      setLobbyError('Display name is required')
      return
    }

    try {
      setLobbyError('')
      const room = await createRoom({ quiz_id: quizID })
      connectToRoom(room.room_id, quizID, displayName)
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : 'Failed to create room'
      setLobbyError(message)
    }
  }

  const handleQuickJoin = async (event: FormEvent) => {
    event.preventDefault()
    if (!quickJoinRoomID.trim()) {
      setLobbyError('Room ID is required')
      return
    }
    if (!displayName.trim()) {
      setLobbyError('Display name is required')
      return
    }

    try {
      setLobbyError('')
      await validateRoom(quickJoinRoomID.trim())
      connectToRoom(quickJoinRoomID.trim(), '', displayName)
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : 'Failed to join room'
      setLobbyError(message)
    }
  }

  const handleStartQuiz = () => {
    wsClientRef.current?.send({ type: 'start_quiz' })
  }

  const handleSubmitAnswer = (answerIndex: number) => {
    setSelectedAnswerIndex(answerIndex)
    if (!question) {
      return
    }
    wsClientRef.current?.send({
      type: 'submit_answer',
      question_id: question.question_id,
      answer_index: answerIndex,
    })
  }

  return (
    <main className="app-shell">
      <header className="app-header">
        <h1>Simple Quiz</h1>
        <p>Realtime quiz lobby and gameplay over WebSocket</p>
      </header>

      {view === 'lobby' ? (
        <section className="panel">
          <h2>Lobby</h2>

          <label className="field-label" htmlFor="display-name">
            Display Name
          </label>
          <input
            id="display-name"
            className="text-input"
            value={displayName}
            onChange={(event) => setDisplayName(event.target.value)}
            placeholder="Enter your name"
          />

          <div className="split-grid">
            <form className="card" onSubmit={handleQuickJoin}>
              <h3>Quick Join</h3>
              <label className="field-label" htmlFor="room-id">
                Room ID
              </label>
              <input
                id="room-id"
                className="text-input"
                value={quickJoinRoomID}
                onChange={(event) => setQuickJoinRoomID(event.target.value)}
                placeholder="e.g. 5f3e2c9a"
              />
              <button type="submit" className="btn-secondary">
                Join Room
              </button>
            </form>
          </div>

          <section className="card">
            <h3>Available Quizzes</h3>
            {loadingQuizzes ? (
              <p>Loading quizzes...</p>
            ) : quizzes.length === 0 ? (
              <p>No quizzes found.</p>
            ) : (
              <ul className="quiz-grid">
                {quizzes.map((quiz) => (
                  <li key={quiz.id} className="quiz-card">
                    <div className="quiz-card-text">
                      <strong>{quiz.title}</strong>
                      <span>{quiz.total_question} questions</span>
                    </div>
                    <button
                      type="button"
                      className="icon-btn"
                      onClick={() => handleCreateRoomForQuiz(quiz.id)}
                      aria-label={`Create room for ${quiz.title}`}
                      title="Create room"
                    >
                      +
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </section>

          {lobbyError ? <p className="error-text">{lobbyError}</p> : null}
        </section>
      ) : (
        <section className="panel">
          <div className="row-between">
            <h2>Room {roomID}</h2>
            <button className="btn-secondary" onClick={resetToLobby} type="button">
              Leave Room
            </button>
          </div>

          <p className="status-line">Connection: {connectionStatus}</p>
          <p className="status-line">
            State: <strong>{sessionState}</strong>
          </p>

          {sessionState === 'waiting' ? (
            <div className="card">
              <h3>Waiting Room</h3>
              {isHost ? (
                <button type="button" className="btn-primary" onClick={handleStartQuiz}>
                  Start Game
                </button>
              ) : (
                <p>Waiting for host to start the game.</p>
              )}
            </div>
          ) : null}

          {sessionState === 'question_active' && question ? (
            <div className="card">
              <h3>
                Question {question.index + 1} / {question.total || totalQuestions}
              </h3>
              <p className="question-text">{question.text}</p>
              <p>Time Left: {secondsLeft}s</p>
              <div className="answers-grid">
                {question.options.map((option, index) => (
                  <button
                    key={`${question.question_id}-${index}`}
                    type="button"
                    className={`answer-btn ${selectedAnswerIndex === index ? 'selected' : ''}`}
                    onClick={() => handleSubmitAnswer(index)}
                    disabled={Boolean(answerAccepted)}
                  >
                    <span>{ANSWER_LABELS[index] ?? String(index + 1)}</span>
                    {option}
                  </button>
                ))}
              </div>
              {!canSubmitAnswer && answerAccepted ? <p>Your answer was submitted.</p> : null}
            </div>
          ) : null}

          {sessionState === 'question_result' && questionResult ? (
            <div className="card">
              <h3>Question Result</h3>
              <p>
                Correct answer: {ANSWER_LABELS[questionResult.correct_answer] ?? questionResult.correct_answer}{' '}
                - {questionResult.correct_text}
              </p>
            </div>
          ) : null}

          {sessionState === 'finished' ? (
            <div className="card">
              <h3>Game Finished</h3>
              <p>Final leaderboard is ready.</p>
            </div>
          ) : null}

          <section className="card">
            <h3>Players</h3>
            {players.length === 0 ? (
              <p>No players yet.</p>
            ) : (
              <ul className="player-list">
                {players.map((player) => (
                  <li key={player.id}>
                    <span>
                      {player.name}
                      {player.is_host ? ' (Host)' : ''}
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </section>

          <section className="card">
            <h3>Leaderboard</h3>
            {leaderBoard?.length === 0 ? (
              <p>Leaderboard will appear as soon as scores are available.</p>
            ) : (
              <ol className="leaderboard-list">
                {leaderBoard?.map((entry) => (
                  <li key={`${entry.name}-${entry.rank}`}>
                    <span>
                      #{entry.rank} {entry.name}
                    </span>
                    <strong>{entry.score}</strong>
                  </li>
                ))}
              </ol>
            )}
          </section>

          {roomError ? <p className="error-text">{roomError}</p> : null}
        </section>
      )}
    </main>
  )
}

export default App
