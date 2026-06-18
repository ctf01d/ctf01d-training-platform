import { useState, useEffect, useCallback } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import * as teamsApi from "../api/teams";
import type { Team, TeamUpdate } from "../api/teams";
import * as membershipsApi from "../api/team-memberships";
import type { TeamMembership, SetRoleRequest } from "../api/team-memberships";
import * as gamesApi from "../api/games";
import * as usersApi from "../api/users";
import type { User } from "../api/users";
import * as universitiesApi from "../api/universities";
import type { University } from "../api/universities";
import * as writeupsApi from "../api/writeups";
import type { Writeup } from "../api/writeups";
import * as scoreboardApi from "../api/scoreboard";
import type { components } from "../api/schema";
import { ErrorDisplay, ActionButton } from "../components/ErrorDisplay";
import { CardBadge } from "../components/Card";
import { FilterSelect } from "../components/FilterSelect";
import {
  DetailHero,
  InfoGroups,
  InfoGroup,
  InfoRow,
  SectionCount,
  renderLink,
  renderLogo,
  formatDateTime,
  safeHref,
} from "../components/DetailInfo";
import { useAuth } from "../auth/AuthContext";

type TeamMembershipEvent = components["schemas"]["TeamMembershipEvent"];

export default function TeamDetailPage() {
  const { id } = useParams<{ id: string }>();
  const teamId = Number(id);
  const navigate = useNavigate();
  const { user, isAdmin } = useAuth();

  const [team, setTeam] = useState<Team | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<{ message?: string } | null>(null);

  const [editing, setEditing] = useState(false);
  const [editForm, setEditForm] = useState<TeamUpdate>({});
  const [saving, setSaving] = useState(false);

  const [members, setMembers] = useState<TeamMembership[]>([]);
  const [joinLoading, setJoinLoading] = useState(false);
  const [inviteUserId, setInviteUserId] = useState("");
  const [inviteLoading, setInviteLoading] = useState(false);

  const [events, setEvents] = useState<TeamMembershipEvent[]>([]);
  const [eventsPage, setEventsPage] = useState(1);
  const [eventsTotal, setEventsTotal] = useState(0);
  const eventsPerPage = 10;

  const [roleForm, setRoleForm] = useState<Record<number, string>>({});
  const [writeups, setWriteups] = useState<Writeup[]>([]);
  const [gameNames, setGameNames] = useState<Record<number, string>>({});
  const [users, setUsers] = useState<Record<number, User>>({});
  const [allUsers, setAllUsers] = useState<User[]>([]);
  const [universityName, setUniversityName] = useState<string | null>(null);
  const [universities, setUniversities] = useState<University[]>([]);
  const [playedGames, setPlayedGames] = useState<
    { gameId: number; name: string; rank?: number; total?: number }[]
  >([]);

  const fetchTeam = useCallback(async () => {
    setLoading(true);
    const { data, error: err } = await teamsApi.getTeam(teamId);
    if (err) setError(err);
    else if (data) setTeam(data);
    setLoading(false);
  }, [teamId]);

  const fetchMembers = useCallback(async () => {
    const { data } = await teamsApi.listTeamMembers(teamId);
    if (data) setMembers(data.items);
  }, [teamId]);

  // Membership history, writeups and the user directory are only available to
  // authenticated viewers; guests browse the public team page without them.
  const fetchEvents = useCallback(async () => {
    if (!user) return;
    const { data } = await teamsApi.listTeamEvents(teamId, {
      page: eventsPage,
      per_page: eventsPerPage,
    });
    if (data) {
      setEvents(data.items);
      setEventsTotal(data.pagination.total);
    }
  }, [teamId, eventsPage, user]);

  const fetchWriteups = useCallback(async () => {
    if (!user) return;
    const { data } = await writeupsApi.listWriteups({ team_id: teamId });
    if (data) {
      setWriteups(data.items);
      const names: Record<number, string> = {};
      const gameIds = Array.from(new Set(data.items.map((w) => w.game_id)));
      for (const gameID of gameIds) {
        const r = await gamesApi.getGame(gameID);
        if (r.data) names[gameID] = r.data.name ?? `Game #${gameID}`;
      }
      setGameNames((prev) => ({ ...prev, ...names }));
    }
  }, [teamId, user]);

  const fetchUsers = useCallback(async () => {
    if (!user) return;
    const { data } = await usersApi.listUsers({ per_page: 100 });
    if (data) {
      setAllUsers(data.items);
      const map: Record<number, User> = {};
      for (const u of data.items) map[u.id] = u;
      setUsers(map);
    }
  }, [user]);

  const fetchUniversities = useCallback(async () => {
    setUniversities(await universitiesApi.listAllUniversities());
  }, []);

  // Games the team has taken part in. Participation is the game roster (a team
  // can be entered without a recorded result), and the placement comes from the
  // game scoreboard endpoint, which is tie-aware and uses final results for
  // finalized games.
  const fetchGames = useCallback(async () => {
    if (!user) return;
    const allGames = await gamesApi.listAllGames();
    const played: typeof playedGames = [];
    const names: Record<number, string> = {};
    for (const g of allGames) {
      const sb = await scoreboardApi.getGameScoreboard(g.id);
      const entry = sb.data?.entries.find((e) => e.team_id === teamId);
      let participated = !!entry;
      if (!participated) {
        const roster = await gamesApi.listGameTeams(g.id);
        participated =
          roster.data?.items.some((gt) => gt.team_id === teamId) ?? false;
      }
      if (!participated) continue;
      names[g.id] = g.name ?? `Game #${g.id}`;
      played.push({
        gameId: g.id,
        name: names[g.id],
        rank: entry?.position,
        total: sb.data ? sb.data.entries.length : undefined,
      });
    }
    setGameNames((prev) => ({ ...prev, ...names }));
    setPlayedGames(played);
  }, [teamId, user]);

  useEffect(() => {
    void fetchTeam();
    void fetchMembers();
    void fetchWriteups();
    void fetchUsers();
    void fetchUniversities();
    void fetchGames();
  }, [
    fetchTeam,
    fetchMembers,
    fetchWriteups,
    fetchUsers,
    fetchUniversities,
    fetchGames,
  ]);

  useEffect(() => {
    void fetchEvents();
  }, [fetchEvents]);

  useEffect(() => {
    if (team?.university_id == null) {
      setUniversityName(null);
      return;
    }
    void universitiesApi.getUniversity(team.university_id).then((r) => {
      if (r.data) setUniversityName(r.data.name ?? null);
    });
  }, [team?.university_id]);

  const userLabel = useCallback(
    (uid: number) =>
      users[uid]?.display_name ?? users[uid]?.user_name ?? `User #${uid}`,
    [users],
  );

  const isManager =
    isAdmin ||
    members.some(
      (m) =>
        m.user_id === user?.id &&
        (m.role === "owner" ||
          m.role === "captain" ||
          m.role === "vice_captain") &&
        m.status === "approved",
    );

  const startEdit = () => {
    if (!team) return;
    setEditForm({
      name: team.name,
      description: team.description ?? undefined,
      website: team.website ?? undefined,
      avatar_url: team.avatar_url ?? undefined,
      university_id: team.university_id ?? undefined,
    });
    setEditing(true);
  };

  const handleSave = async () => {
    setSaving(true);
    const { data, error: err } = await teamsApi.updateTeam(teamId, editForm);
    setSaving(false);
    if (err) {
      setError(err);
      return;
    }
    if (data) {
      setTeam(data);
      setEditing(false);
    }
  };

  const handleJoin = async () => {
    setJoinLoading(true);
    const { error: err } = await teamsApi.requestJoinTeam(teamId);
    setJoinLoading(false);
    if (err) {
      setError(err);
      return;
    }
    await fetchMembers();
  };

  const handleInvite = async (e: React.FormEvent) => {
    e.preventDefault();
    const uid = Number(inviteUserId);
    if (!uid) return;
    setInviteLoading(true);
    const { error: err } = await teamsApi.inviteToTeam(teamId, uid);
    setInviteLoading(false);
    if (err) {
      setError(err);
      return;
    }
    setInviteUserId("");
    await fetchMembers();
  };

  const handleMembershipAction = async (
    action: () => Promise<{ data?: unknown; error?: { message: string } }>,
  ) => {
    const { error: err } = await action();
    if (err) {
      setError(err);
      return;
    }
    await fetchMembers();
  };

  const handleSetRole = async (membershipId: number, role: string) => {
    const { error: err } = await membershipsApi.setTeamMembershipRole(
      membershipId,
      { role: role as SetRoleRequest["role"] },
    );
    if (err) {
      setError(err);
      return;
    }
    await fetchMembers();
  };

  const handleDelete = async () => {
    const { error: err } = await teamsApi.deleteTeam(teamId);
    if (err) {
      setError(err);
      return;
    }
    navigate("/teams");
  };

  const handleDeleteWriteup = async (writeupId: number) => {
    const { error: err } = await writeupsApi.deleteWriteup(writeupId);
    if (err) {
      setError(err);
      return;
    }
    await fetchWriteups();
  };

  if (loading) return <div className="loading">Loading...</div>;
  if (!team) return <ErrorDisplay error={error} onRetry={fetchTeam} />;

  const myMembership = members.find((m) => m.user_id === user?.id);
  const approvedCount = members.filter((m) => m.status === "approved").length;
  const memberUserIds = new Set(members.map((m) => m.user_id));
  const invitableUsers = allUsers.filter((u) => !memberUserIds.has(u.id));

  const universityNode =
    team.university_id != null ? (
      <Link to={`/universities/${team.university_id}`}>
        {universityName ?? `University #${team.university_id}`}
      </Link>
    ) : (
      "—"
    );

  return (
    <div className="page detail-page">
      <ErrorDisplay error={error} />

      {!editing ? (
        <>
          <DetailHero
            kicker={`Team #${team.id}`}
            title={team.name}
            avatarUrl={team.avatar_url}
            avatarText={team.name}
            badges={
              myMembership ? (
                <>
                  <CardBadge variant={myMembership.role}>
                    {myMembership.role}
                  </CardBadge>
                  <CardBadge variant={myMembership.status}>
                    {myMembership.status}
                  </CardBadge>
                </>
              ) : undefined
            }
            summary={[
              {
                label: "Members",
                value: `${approvedCount} approved · ${members.length} total`,
              },
              {
                label: "Captain",
                value: team.captain_id ? userLabel(team.captain_id) : "—",
              },
              { label: "University", value: universityNode },
            ]}
            actions={
              <>
                <button
                  className="btn btn-sm"
                  onClick={() => navigate("/teams")}
                >
                  Back
                </button>
                {team.website && (
                  <a
                    className="btn btn-sm"
                    href={safeHref(team.website)}
                    target="_blank"
                    rel="noreferrer"
                  >
                    Website
                  </a>
                )}
                {isManager && (
                  <button
                    className="btn btn-sm btn-primary"
                    onClick={startEdit}
                  >
                    Edit
                  </button>
                )}
                {user && !myMembership && !isManager && (
                  <button
                    className="btn btn-sm btn-primary"
                    onClick={() => void handleJoin()}
                    disabled={joinLoading}
                  >
                    {joinLoading ? "Requesting..." : "Request to Join"}
                  </button>
                )}
                {isManager && (
                  <ActionButton
                    onClick={handleDelete}
                    variant="danger"
                    confirm="Delete this team?"
                  >
                    Delete
                  </ActionButton>
                )}
              </>
            }
          />

          <div className="detail-section">
            <div className="section-head">
              <h3>Team Info</h3>
            </div>
            <InfoGroups>
              <InfoGroup title="About">
                <InfoRow label="Description">{team.description ?? "—"}</InfoRow>
                <InfoRow label="Website">{renderLink(team.website)}</InfoRow>
                <InfoRow label="Avatar">{renderLogo(team.avatar_url)}</InfoRow>
              </InfoGroup>

              <InfoGroup title="Details">
                <InfoRow label="Captain">
                  {team.captain_id ? userLabel(team.captain_id) : "—"}
                </InfoRow>
                <InfoRow label="University">{universityNode}</InfoRow>
              </InfoGroup>
            </InfoGroups>
          </div>
        </>
      ) : (
        <form
          onSubmit={(e) => {
            e.preventDefault();
            void handleSave();
          }}
          className="edit-form"
        >
          <div className="form-group">
            <label>Name</label>
            <input
              value={editForm.name ?? ""}
              onChange={(e) =>
                setEditForm((f) => ({ ...f, name: e.target.value }))
              }
            />
          </div>
          <div className="form-group">
            <label>Description</label>
            <input
              value={editForm.description ?? ""}
              onChange={(e) =>
                setEditForm((f) => ({ ...f, description: e.target.value }))
              }
            />
          </div>
          <div className="form-group">
            <label>Website</label>
            <input
              value={editForm.website ?? ""}
              onChange={(e) =>
                setEditForm((f) => ({ ...f, website: e.target.value }))
              }
            />
          </div>
          <div className="form-group">
            <label>Avatar URL</label>
            <input
              value={editForm.avatar_url ?? ""}
              onChange={(e) =>
                setEditForm((f) => ({ ...f, avatar_url: e.target.value }))
              }
            />
          </div>
          <div className="form-group">
            <label>University</label>
            <FilterSelect
              placeholder="Search universities…"
              allowClear
              value={editForm.university_id ?? null}
              onChange={(id) =>
                setEditForm((f) => ({ ...f, university_id: id }))
              }
              options={universities.map((u) => ({
                id: u.id,
                label: u.name ?? `University #${u.id}`,
              }))}
            />
          </div>
          <div className="form-actions">
            <button type="submit" className="btn btn-primary" disabled={saving}>
              {saving ? "Saving..." : "Save"}
            </button>
            <button
              type="button"
              className="btn"
              onClick={() => setEditing(false)}
            >
              Cancel
            </button>
          </div>
        </form>
      )}

      <div className="detail-section">
        <div className="section-head">
          <h3>
            Members <SectionCount n={members.length} />
          </h3>
        </div>
        {members.length > 0 ? (
          <table className="data-table">
            <thead>
              <tr>
                <th>Member</th>
                <th>Role</th>
                <th>Status</th>
                {isManager && <th></th>}
              </tr>
            </thead>
            <tbody>
              {members.map((m) => (
                <tr key={m.id}>
                  <td>
                    <Link to={`/users/${m.user_id}`}>
                      {userLabel(m.user_id)}
                    </Link>
                  </td>
                  <td>
                    {isManager ? (
                      <select
                        value={roleForm[m.id] ?? m.role}
                        onChange={(e) =>
                          setRoleForm((prev) => ({
                            ...prev,
                            [m.id]: e.target.value,
                          }))
                        }
                        onBlur={() => {
                          const newRole = roleForm[m.id];
                          if (newRole && newRole !== m.role)
                            void handleSetRole(m.id, newRole);
                        }}
                      >
                        {(
                          [
                            "owner",
                            "captain",
                            "vice_captain",
                            "player",
                            "guest",
                          ] as const
                        ).map((r) => (
                          <option key={r} value={r}>
                            {r}
                          </option>
                        ))}
                      </select>
                    ) : (
                      <CardBadge variant={m.role}>{m.role}</CardBadge>
                    )}
                  </td>
                  <td>
                    <CardBadge variant={m.status}>{m.status}</CardBadge>
                  </td>
                  {isManager && (
                    <td className="actions-cell">
                      {m.status === "pending" && (
                        <>
                          <ActionButton
                            onClick={() =>
                              void handleMembershipAction(() =>
                                membershipsApi.approveTeamMembership(m.id),
                              )
                            }
                            variant="success"
                          >
                            Approve
                          </ActionButton>
                          <ActionButton
                            onClick={() =>
                              void handleMembershipAction(() =>
                                membershipsApi.rejectTeamMembership(m.id),
                              )
                            }
                            variant="danger"
                          >
                            Reject
                          </ActionButton>
                        </>
                      )}
                      {m.user_id === user?.id && m.status === "pending" && (
                        <>
                          <ActionButton
                            onClick={() =>
                              void handleMembershipAction(() =>
                                membershipsApi.acceptTeamMembership(m.id),
                              )
                            }
                            variant="success"
                          >
                            Accept
                          </ActionButton>
                          <ActionButton
                            onClick={() =>
                              void handleMembershipAction(() =>
                                membershipsApi.declineTeamMembership(m.id),
                              )
                            }
                            variant="danger"
                          >
                            Decline
                          </ActionButton>
                        </>
                      )}
                      <ActionButton
                        onClick={() =>
                          void handleMembershipAction(() =>
                            membershipsApi.deleteTeamMembership(m.id),
                          )
                        }
                        variant="danger"
                        confirm="Remove this member?"
                      >
                        Remove
                      </ActionButton>
                    </td>
                  )}
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <p className="section-empty">No members.</p>
        )}

        {isManager && (
          <form onSubmit={(e) => void handleInvite(e)} className="inline-form">
            <select
              value={inviteUserId}
              onChange={(e) => setInviteUserId(e.target.value)}
              required
            >
              <option value="">Invite user…</option>
              {invitableUsers.map((u) => (
                <option key={u.id} value={u.id}>
                  {u.display_name} ({u.user_name})
                </option>
              ))}
            </select>
            <button
              type="submit"
              className="btn btn-sm"
              disabled={inviteLoading}
            >
              {inviteLoading ? "Inviting..." : "Invite"}
            </button>
          </form>
        )}
      </div>

      {user && (
        <>
          <div className="detail-section">
            <div className="section-head">
              <h3>
                Games <SectionCount n={playedGames.length} />
              </h3>
            </div>
            {playedGames.length > 0 ? (
              <InfoGroups>
                <InfoGroup title="Played">
                  {playedGames.map((g) => (
                    <InfoRow
                      key={g.gameId}
                      label={
                        <Link to={`/games/${g.gameId}`}>{g.name}</Link>
                      }
                    >
                      {g.rank != null && g.total != null ? (
                        <span
                          className="score-cell"
                          title={`Placed ${g.rank} of ${g.total} teams`}
                        >
                          {g.rank} / {g.total}
                        </span>
                      ) : (
                        <span className="muted-dash">—</span>
                      )}
                    </InfoRow>
                  ))}
                </InfoGroup>
              </InfoGroups>
            ) : (
              <p className="section-empty">No games played yet.</p>
            )}
          </div>

          <div className="detail-section">
            <div className="section-head">
              <h3>
                Writeups <SectionCount n={writeups.length} />
              </h3>
            </div>
            {writeups.length > 0 ? (
              <table className="data-table">
                <thead>
                  <tr>
                    <th>Game</th>
                    <th>Title</th>
                    <th>Link</th>
                    {isManager && <th></th>}
                  </tr>
                </thead>
                <tbody>
                  {writeups.map((w) => (
                    <tr key={w.id}>
                      <td>
                        <Link to={`/games/${w.game_id}`}>
                          {gameNames[w.game_id] ?? `Game #${w.game_id}`}
                        </Link>
                      </td>
                      <td>{w.title}</td>
                      <td>
                        <a
                          href={safeHref(w.url)}
                          target="_blank"
                          rel="noreferrer"
                        >
                          Open ↗
                        </a>
                      </td>
                      {isManager && (
                        <td className="actions-cell">
                          <ActionButton
                            onClick={() => void handleDeleteWriteup(w.id)}
                            variant="danger"
                            confirm="Delete this writeup?"
                          >
                            Delete
                          </ActionButton>
                        </td>
                      )}
                    </tr>
                  ))}
                </tbody>
              </table>
            ) : (
              <p className="section-empty">No writeups.</p>
            )}
          </div>

          <div className="detail-section">
            <div className="section-head">
              <h3>
                Events <SectionCount n={eventsTotal} />
              </h3>
            </div>
            {events.length > 0 ? (
              <>
                <table className="data-table">
                  <thead>
                    <tr>
                      <th>Date</th>
                      <th>Member</th>
                      <th>Action</th>
                      <th>Role change</th>
                      <th>Status change</th>
                    </tr>
                  </thead>
                  <tbody>
                    {events.map((ev) => (
                      <tr key={ev.id}>
                        <td>{formatDateTime(ev.created_at)}</td>
                        <td>{userLabel(ev.user_id)}</td>
                        <td>{ev.action}</td>
                        <td>
                          {ev.from_role && ev.to_role
                            ? `${ev.from_role} → ${ev.to_role}`
                            : "—"}
                        </td>
                        <td>
                          {ev.from_status && ev.to_status
                            ? `${ev.from_status} → ${ev.to_status}`
                            : "—"}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
                {Math.ceil(eventsTotal / eventsPerPage) > 1 && (
                  <div className="pagination">
                    <button
                      className="btn btn-sm"
                      disabled={eventsPage <= 1}
                      onClick={() => setEventsPage(eventsPage - 1)}
                    >
                      Prev
                    </button>
                    <span>
                      Page {eventsPage} of{" "}
                      {Math.ceil(eventsTotal / eventsPerPage)}
                    </span>
                    <button
                      className="btn btn-sm"
                      disabled={
                        eventsPage >= Math.ceil(eventsTotal / eventsPerPage)
                      }
                      onClick={() => setEventsPage(eventsPage + 1)}
                    >
                      Next
                    </button>
                  </div>
                )}
              </>
            ) : (
              <p className="section-empty">No events.</p>
            )}
          </div>
        </>
      )}
    </div>
  );
}
