import { useState, useEffect, useCallback, type CSSProperties } from "react";
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
import { usePageTitle } from "../components/usePageTitle";
import {
  DetailHero,
  InfoGroups,
  InfoGroup,
  InfoRow,
  SectionCount,
  formatDateTime,
  safeHref,
} from "../components/DetailInfo";
import { useAuth } from "../auth/AuthContext";
import { useI18n } from "../i18n/I18nContext";

type TeamMembershipEvent = components["schemas"]["TeamMembershipEvent"];

export default function TeamDetailPage() {
  const { t } = useI18n();
  const { id } = useParams<{ id: string }>();
  const teamId = Number(id);
  const navigate = useNavigate();
  const { user, isAdmin } = useAuth();

  const [team, setTeam] = useState<Team | null>(null);
  usePageTitle(team?.name);
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
        if (r.data) names[gameID] = r.data.name ?? `${t("Game")} #${gameID}`;
      }
      setGameNames((prev) => ({ ...prev, ...names }));
    }
  }, [teamId, t, user]);

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
      names[g.id] = g.name ?? `${t("Game")} #${g.id}`;
      played.push({
        gameId: g.id,
        name: names[g.id],
        rank: entry?.position,
        total: sb.data ? sb.data.entries.length : undefined,
      });
    }
    setGameNames((prev) => ({ ...prev, ...names }));
    setPlayedGames(played);
  }, [teamId, t, user]);

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
      users[uid]?.display_name ??
      users[uid]?.user_name ??
      `${t("User")} #${uid}`,
    [t, users],
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

  // The actions column is also needed for a non-manager whose own membership is
  // pending (an invite), so they can Accept/Decline it from the team page.
  const showMemberActions =
    isManager ||
    members.some((m) => m.user_id === user?.id && m.status === "pending");

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

  if (loading) return <div className="loading">{t("Loading...")}</div>;
  if (!team) return <ErrorDisplay error={error} onRetry={fetchTeam} />;

  const myMembership = members.find((m) => m.user_id === user?.id);
  const approvedCount = members.filter((m) => m.status === "approved").length;
  const memberUserIds = new Set(members.map((m) => m.user_id));
  const invitableUsers = allUsers.filter((u) => !memberUserIds.has(u.id));
  const hasTeamInfo = Boolean(team.description);

  const universityNode =
    team.university_id != null ? (
      <Link to={`/universities/${team.university_id}`}>
        {universityName ?? `${t("University")} #${team.university_id}`}
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
            kicker={`${t("Team")} #${team.id}`}
            title={team.name}
            avatarUrl={team.avatar_url}
            avatarText={team.name}
            badges={
              myMembership ? (
                <>
                  <CardBadge variant={myMembership.role}>
                    {t(myMembership.role)}
                  </CardBadge>
                  <CardBadge variant={myMembership.status}>
                    {t(myMembership.status)}
                  </CardBadge>
                </>
              ) : undefined
            }
            summary={[
              {
                label: t("Members"),
                value: t("{approved} approved · {total} total", {
                  approved: approvedCount,
                  total: members.length,
                }),
              },
              {
                label: t("Captain"),
                value: team.captain_id ? userLabel(team.captain_id) : "—",
              },
              { label: t("University"), value: universityNode },
            ]}
            actions={
              <>
                <button
                  className="btn btn-sm"
                  onClick={() => navigate("/teams")}
                >
                  {t("Back")}
                </button>
                {team.website && (
                  <a
                    className="btn btn-sm"
                    href={safeHref(team.website)}
                    target="_blank"
                    rel="noreferrer"
                  >
                    {t("Website")}
                  </a>
                )}
                {isManager && (
                  <button
                    className="btn btn-sm btn-primary"
                    onClick={startEdit}
                  >
                    {t("Edit")}
                  </button>
                )}
                {user && !myMembership && !isManager && (
                  <button
                    className="btn btn-sm btn-primary"
                    onClick={() => void handleJoin()}
                    disabled={joinLoading}
                  >
                    {joinLoading ? t("Requesting...") : t("Request to Join")}
                  </button>
                )}
                {isManager && (
                  <ActionButton
                    onClick={handleDelete}
                    variant="danger"
                    confirm={t("Delete this team?")}
                  >
                    {t("Delete")}
                  </ActionButton>
                )}
              </>
            }
          />

          {hasTeamInfo && (
            <div className="detail-section">
              <div className="section-head">
                <h3>{t("Team Info")}</h3>
              </div>
              <InfoGroups className="team-info-groups">
                <InfoGroup title={t("About")} className="team-info-about">
                  <InfoRow label={t("Description")}>{team.description}</InfoRow>
                </InfoGroup>
              </InfoGroups>
            </div>
          )}
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
            <label>{t("Name")}</label>
            <input
              value={editForm.name ?? ""}
              onChange={(e) =>
                setEditForm((f) => ({ ...f, name: e.target.value }))
              }
            />
          </div>
          <div className="form-group">
            <label>{t("Description")}</label>
            <input
              value={editForm.description ?? ""}
              onChange={(e) =>
                setEditForm((f) => ({ ...f, description: e.target.value }))
              }
            />
          </div>
          <div className="form-group">
            <label>{t("Website")}</label>
            <input
              value={editForm.website ?? ""}
              onChange={(e) =>
                setEditForm((f) => ({ ...f, website: e.target.value }))
              }
            />
          </div>
          <div className="form-group">
            <label>{t("Avatar URL")}</label>
            <input
              value={editForm.avatar_url ?? ""}
              onChange={(e) =>
                setEditForm((f) => ({ ...f, avatar_url: e.target.value }))
              }
            />
          </div>
          <div className="form-group">
            <label>{t("University")}</label>
            <FilterSelect
              placeholder={t("Search universities...")}
              allowClear
              value={editForm.university_id ?? null}
              onChange={(id) =>
                setEditForm((f) => ({ ...f, university_id: id }))
              }
              options={universities.map((u) => ({
                id: u.id,
                label: u.name ?? `${t("University")} #${u.id}`,
              }))}
            />
          </div>
          <div className="form-actions">
            <button type="submit" className="btn btn-primary" disabled={saving}>
              {saving ? t("Saving...") : t("Save")}
            </button>
            <button
              type="button"
              className="btn"
              onClick={() => setEditing(false)}
            >
              {t("Cancel")}
            </button>
          </div>
        </form>
      )}

      <div className="detail-section">
        <div className="section-head">
          <h3>
            {t("Members")} <SectionCount n={members.length} />
          </h3>
        </div>
        {members.length > 0 ? (
          <table className="data-table">
            <thead>
              <tr>
                <th>{t("Member")}</th>
                <th>{t("Role")}</th>
                <th>{t("Status")}</th>
                {showMemberActions && <th></th>}
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
                            {t(r)}
                          </option>
                        ))}
                      </select>
                    ) : (
                      <CardBadge variant={m.role}>{t(m.role)}</CardBadge>
                    )}
                  </td>
                  <td>
                    <CardBadge variant={m.status}>{t(m.status)}</CardBadge>
                  </td>
                  {showMemberActions && (
                    <td className="actions-cell">
                      {isManager && m.status === "pending" && (
                        <>
                          <ActionButton
                            onClick={() =>
                              void handleMembershipAction(() =>
                                membershipsApi.approveTeamMembership(m.id),
                              )
                            }
                            variant="success"
                          >
                            {t("Approve")}
                          </ActionButton>
                          <ActionButton
                            onClick={() =>
                              void handleMembershipAction(() =>
                                membershipsApi.rejectTeamMembership(m.id),
                              )
                            }
                            variant="danger"
                          >
                            {t("Reject")}
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
                            {t("Accept")}
                          </ActionButton>
                          <ActionButton
                            onClick={() =>
                              void handleMembershipAction(() =>
                                membershipsApi.declineTeamMembership(m.id),
                              )
                            }
                            variant="danger"
                          >
                            {t("Decline")}
                          </ActionButton>
                        </>
                      )}
                      {isManager && (
                        <ActionButton
                          onClick={() =>
                            void handleMembershipAction(() =>
                              membershipsApi.deleteTeamMembership(m.id),
                            )
                          }
                          variant="danger"
                          confirm={t("Remove this member?")}
                        >
                          {t("Remove")}
                        </ActionButton>
                      )}
                    </td>
                  )}
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <p className="section-empty">{t("No members.")}</p>
        )}

        {isManager && (
          <form onSubmit={(e) => void handleInvite(e)} className="inline-form">
            <select
              value={inviteUserId}
              onChange={(e) => setInviteUserId(e.target.value)}
              required
            >
              <option value="">{t("Invite user...")}</option>
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
              {inviteLoading ? t("Inviting...") : t("Invite")}
            </button>
          </form>
        )}
      </div>

      {user && (
        <>
          <div className="detail-section">
            <div className="section-head">
              <h3>
                {t("Games")} <SectionCount n={playedGames.length} />
              </h3>
            </div>
            {playedGames.length > 0 ? (
              <div className="team-games-list">
                {playedGames.map((g) => {
                  const rank = g.rank;
                  const total = g.total;
                  const hasPlacement = rank != null && total != null;
                  const performance = hasPlacement
                    ? Math.max(
                        0,
                        Math.min(100, ((total - rank + 1) / total) * 100),
                      )
                    : 0;
                  const meterStyle = {
                    "--value": `${performance}%`,
                  } as CSSProperties;
                  const podiumClass =
                    hasPlacement && rank >= 1 && rank <= 3
                      ? ` podium-${rank}`
                      : "";

                  return (
                    <div
                      className={`team-game-row${podiumClass}`}
                      key={g.gameId}
                    >
                      <div className="team-game-main">
                        <Link to={`/games/${g.gameId}`}>{g.name}</Link>
                        <span
                          className="score-cell"
                          title={
                            hasPlacement
                              ? t("Placed {rank} of {total} teams", {
                                  rank,
                                  total,
                                })
                              : undefined
                          }
                        >
                          {hasPlacement ? (
                            `${rank} / ${total}`
                          ) : (
                            <span className="muted-dash">—</span>
                          )}
                        </span>
                      </div>
                      <div
                        className="team-game-meter"
                        style={meterStyle}
                        aria-hidden="true"
                      />
                    </div>
                  );
                })}
              </div>
            ) : (
              <p className="section-empty">{t("No games played yet.")}</p>
            )}
          </div>

          <div className="detail-section">
            <div className="section-head">
              <h3>
                {t("Writeups")} <SectionCount n={writeups.length} />
              </h3>
            </div>
            {writeups.length > 0 ? (
              <table className="data-table">
                <thead>
                  <tr>
                    <th>{t("Game")}</th>
                    <th>{t("Title")}</th>
                    <th>{t("Link")}</th>
                    {isManager && <th></th>}
                  </tr>
                </thead>
                <tbody>
                  {writeups.map((w) => (
                    <tr key={w.id}>
                      <td>
                        <Link to={`/games/${w.game_id}`}>
                          {gameNames[w.game_id] ?? `${t("Game")} #${w.game_id}`}
                        </Link>
                      </td>
                      <td>{w.title}</td>
                      <td>
                        <a
                          href={safeHref(w.url)}
                          target="_blank"
                          rel="noreferrer"
                        >
                          {t("Open ↗")}
                        </a>
                      </td>
                      {isManager && (
                        <td className="actions-cell">
                          <ActionButton
                            onClick={() => void handleDeleteWriteup(w.id)}
                            variant="danger"
                            confirm={t("Delete this writeup?")}
                          >
                            {t("Delete")}
                          </ActionButton>
                        </td>
                      )}
                    </tr>
                  ))}
                </tbody>
              </table>
            ) : (
              <p className="section-empty">{t("No writeups.")}</p>
            )}
          </div>

          <div className="detail-section">
            <div className="section-head">
              <h3>
                {t("Events")} <SectionCount n={eventsTotal} />
              </h3>
            </div>
            {events.length > 0 ? (
              <>
                <table className="data-table">
                  <thead>
                    <tr>
                      <th>{t("Date")}</th>
                      <th>{t("Member")}</th>
                      <th>{t("Action")}</th>
                      <th>{t("Role change")}</th>
                      <th>{t("Status change")}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {events.map((ev) => (
                      <tr key={ev.id}>
                        <td>{formatDateTime(ev.created_at)}</td>
                        <td>{userLabel(ev.user_id)}</td>
                        <td>{t(ev.action)}</td>
                        <td>
                          {ev.from_role && ev.to_role
                            ? `${t(ev.from_role)} → ${t(ev.to_role)}`
                            : "—"}
                        </td>
                        <td>
                          {ev.from_status && ev.to_status
                            ? `${t(ev.from_status)} → ${t(ev.to_status)}`
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
                      {t("Prev")}
                    </button>
                    <span>
                      {t("Page")} {eventsPage} {t("of")}{" "}
                      {Math.ceil(eventsTotal / eventsPerPage)}
                    </span>
                    <button
                      className="btn btn-sm"
                      disabled={
                        eventsPage >= Math.ceil(eventsTotal / eventsPerPage)
                      }
                      onClick={() => setEventsPage(eventsPage + 1)}
                    >
                      {t("Next")}
                    </button>
                  </div>
                )}
              </>
            ) : (
              <p className="section-empty">{t("No events.")}</p>
            )}
          </div>
        </>
      )}
    </div>
  );
}
