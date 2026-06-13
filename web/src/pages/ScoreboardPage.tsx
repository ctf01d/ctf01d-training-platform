import { useState, useEffect, useCallback } from 'react'
import * as scoreboardApi from '../api/scoreboard'
import * as gamesApi from '../api/games'
import type { Game } from '../api/games'
import type { components } from '../api/schema'
import { ErrorDisplay } from '../components/ErrorDisplay'
import { CardBadge } from '../components/Card'

type ScoreboardEntry = components['schemas']['ScoreboardEntry']
type GlobalScoreboard = components['schemas']['GlobalScoreboard']
type Scoreboard = components['schemas']['Scoreboard']

function StatusBadge({ status }: { status: string }) {
  return <CardBadge variant={status}>{status}</CardBadge>
}

const fmtScore = (score: number) => score.toLocaleString()

export default function ScoreboardPage() {
  const [tab, setTab] = useState<'global' | 'game'>('global')

  const [globalEntries, setGlobalEntries] = useState<GlobalScoreboard['entries']>([])
  const [globalLoading, setGlobalLoading] = useState(true)
  const [globalError, setGlobalError] = useState<{ message?: string } | null>(null)

  const [games, setGames] = useState<Game[]>([])
  const [selectedGameId, setSelectedGameId] = useState('')
  const [gameScoreboard, setGameScoreboard] = useState<Scoreboard | null>(null)
  const [gameLoading, setGameLoading] = useState(false)
  const [gameError, setGameError] = useState<{ message?: string } | null>(null)

  const fetchGlobal = useCallback(async () => {
    setGlobalLoading(true)
    const { data, error: err } = await scoreboardApi.getGlobalScoreboard()
    if (err) setGlobalError(err)
    else if (data) setGlobalEntries(data.entries)
    setGlobalLoading(false)
  }, [])

  const fetchGames = useCallback(async () => {
    const { data } = await gamesApi.listGames({ page: 1, per_page: 100 })
    if (data) setGames(data.items)
  }, [])

  useEffect(() => {
    void fetchGlobal()
    void fetchGames()
  }, [fetchGlobal, fetchGames])

  const fetchGameScoreboard = useCallback(async () => {
    const gid = Number(selectedGameId)
    if (!gid) { setGameScoreboard(null); return }
    setGameLoading(true)
    setGameError(null)
    const { data, error: err } = await scoreboardApi.getGameScoreboard(gid)
    if (err) setGameError(err)
    else if (data) setGameScoreboard(data)
    setGameLoading(false)
  }, [selectedGameId])

  useEffect(() => {
    if (selectedGameId) void fetchGameScoreboard()
    else setGameScoreboard(null)
  }, [selectedGameId, fetchGameScoreboard])

  return (
    <div className="page">
      <div className="page-header">
        <h1>Scoreboard</h1>
      </div>

      <div className="tabs" role="tablist" aria-label="Scoreboard scope">
        <button className={`btn btn-sm ${tab === 'global' ? 'btn-primary' : ''}`} onClick={() => setTab('global')}>Global</button>
        <button className={`btn btn-sm ${tab === 'game' ? 'btn-primary' : ''}`} onClick={() => setTab('game')}>By Game</button>
      </div>

      {tab === 'global' && (
        <div className="detail-section">
          <h3>Global Scoreboard</h3>
          <ErrorDisplay error={globalError} onRetry={fetchGlobal} />
          {globalLoading ? (
            <div className="loading">Loading...</div>
          ) : globalEntries.length === 0 ? (
            <div className="empty-state">No scoreboard entries</div>
          ) : (
            <div className="table-shell table-shell-scroll">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>Position</th>
                    <th>Team</th>
                    <th className="numeric">Total Score</th>
                  </tr>
                </thead>
                <tbody>
                  {globalEntries
                    .sort((a, b) => b.total_score - a.total_score)
                    .map((entry, idx) => (
                      <tr key={entry.team_id}>
                        <td className="rank-cell">{idx + 1}</td>
                        <td>{entry.team_name}</td>
                        <td className="numeric score-cell">{fmtScore(entry.total_score)}</td>
                      </tr>
                    ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      {tab === 'game' && (
        <div className="detail-section">
          <h3>Game Scoreboard</h3>
          <div className="form-group">
            <select value={selectedGameId} onChange={e => setSelectedGameId(e.target.value)}>
              <option value="">Select a game...</option>
              {games.map(g => (
                <option key={g.id} value={g.id}>{g.name ?? `Game #${g.id}`}</option>
              ))}
            </select>
          </div>

          <ErrorDisplay error={gameError} />

          {gameLoading ? (
            <div className="loading">Loading...</div>
          ) : gameScoreboard ? (
            <>
              <div className="status-line">
                <span>Status:</span>
                <StatusBadge status={gameScoreboard.status} />
              </div>
              {gameScoreboard.entries.length === 0 ? (
                <div className="empty-state">No entries for this game</div>
              ) : (
                <div className="table-shell table-shell-scroll">
                  <table className="data-table">
                    <thead>
                      <tr>
                        <th>Position</th>
                        <th>Team</th>
                        <th className="numeric">Score</th>
                      </tr>
                    </thead>
                    <tbody>
                      {gameScoreboard.entries
                        .sort((a, b) => a.position - b.position)
                        .map((entry: ScoreboardEntry) => (
                          <tr key={entry.team_id}>
                            <td className="rank-cell">{entry.position}</td>
                            <td>{entry.team_name}</td>
                            <td className="numeric score-cell">{fmtScore(entry.score)}</td>
                          </tr>
                        ))}
                    </tbody>
                  </table>
                </div>
              )}
            </>
          ) : selectedGameId ? null : (
            <div className="empty-state">Select a game to view its scoreboard</div>
          )}
        </div>
      )}
    </div>
  )
}
