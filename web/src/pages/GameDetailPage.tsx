import { useState, useEffect, useCallback } from "react";
import { Link, useParams, useNavigate } from "react-router-dom";
import * as gamesApi from "../api/games";
import type { Game, GameUpdate } from "../api/games";
import * as gameTeamsApi from "../api/game-teams";
import type { GameTeam, GameTeamCreate } from "../api/game-teams";
import * as servicesApi from "../api/services";
import type { Service } from "../api/services";
import * as resultsApi from "../api/results";
import type { Result, ResultCreate } from "../api/results";
import * as writeupsApi from "../api/writeups";
import type { Writeup, WriteupCreate } from "../api/writeups";
import * as teamsApi from "../api/teams";
import type { Team } from "../api/teams";
import {
  ErrorDisplay,
  ActionButton,
  handleApiError,
} from "../components/ErrorDisplay";
import { CardBadge } from "../components/Card";
import { TeamLink } from "../components/TeamLink";
import { FilterSelect } from "../components/FilterSelect";
import { usePageTitle } from "../components/usePageTitle";
import {
  DetailHero,
  InfoGroups,
  InfoGroup,
  InfoRow,
  SectionCount,
  RelativeTime,
  Duration,
  renderLink,
  formatDateTime as formatDate,
  safeHref,
} from "../components/DetailInfo";
import { useAuth } from "../auth/AuthContext";
import {
  datetimeLocalToRFC3339,
  rfc3339ToDatetimeLocal,
} from "../api/datetime";
import { useI18n } from "../i18n/I18nContext";

export default function GameDetailPage() {
  const { t } = useI18n();
  const { id } = useParams<{ id: string }>();
  const gameId = Number(id);
  const navigate = useNavigate();
  const { user, isPlayer, isAdmin } = useAuth();

  const [game, setGame] = useState<Game | null>(null);
  usePageTitle(game?.name);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<{ message?: string } | null>(null);

  const [editing, setEditing] = useState(false);
  const [editForm, setEditForm] = useState<GameUpdate>({});
  const [saving, setSaving] = useState(false);

  const [gameTeams, setGameTeams] = useState<GameTeam[]>([]);
  const [teamNames, setTeamNames] = useState<Record<number, string>>({});
  const [manageableTeamIds, setManageableTeamIds] = useState<number[]>([]);
  const [serviceIds, setServiceIds] = useState<number[]>([]);
  const [serviceDetails, setServiceDetails] = useState<Record<number, Service>>(
    {},
  );
  const [results, setResults] = useState<Result[]>([]);
  const [writeups, setWriteups] = useState<Writeup[]>([]);
  const [allTeams, setAllTeams] = useState<Team[]>([]);
  const [allServices, setAllServices] = useState<Service[]>([]);
  const [organizerTeam, setOrganizerTeam] = useState<Team | null>(null);

  const [addTeamForm, setAddTeamForm] = useState<GameTeamCreate>({
    game_id: gameId,
    team_id: 0,
  });
  const [addServiceId, setAddServiceId] = useState("");
  const [addResultForm, setAddResultForm] = useState<ResultCreate>({
    game_id: gameId,
    team_id: 0,
    score: 0,
  });
  const [addWriteupForm, setAddWriteupForm] = useState<WriteupCreate>({
    game_id: gameId,
    team_id: 0,
    title: "",
    url: "https://",
  });

  const fetchGame = useCallback(async () => {
    setLoading(true);
    const { data, error: err } = await gamesApi.getGame(gameId);
    if (err) setError(err);
    else if (data) setGame(data);
    setLoading(false);
  }, [gameId]);

  const fetchGameTeams = useCallback(async () => {
    const { data } = await gamesApi.listGameTeams(gameId);
    if (data) {
      setGameTeams(data.items);
      const ids = data.items.map((gt) => gt.team_id);
      const nameMap: Record<number, string> = {};
      const manageable: number[] = [];
      for (const tid of ids) {
        const r = await teamsApi.getTeam(tid);
        if (r.data) nameMap[tid] = r.data.name;
        if (isAdmin) {
          manageable.push(tid);
        } else if (user) {
          const r = await teamsApi.listTeamMembers(tid);
          const membership = r.data?.items.find((m) => m.user_id === user.id);
          if (
            membership?.status === "approved" &&
            (membership.role === "owner" ||
              membership.role === "captain" ||
              membership.role === "vice_captain")
          ) {
            manageable.push(tid);
          }
        }
      }
      setTeamNames((prev) => {
        let changed = false;
        const next = { ...prev };
        for (const [tid, name] of Object.entries(nameMap)) {
          const teamId = Number(tid);
          if (next[teamId] !== name) {
            next[teamId] = name;
            changed = true;
          }
        }
        return changed ? next : prev;
      });
      setManageableTeamIds(manageable);
    }
  }, [gameId, isAdmin, user]);

  const fetchServices = useCallback(async () => {
    const { data } = await gamesApi.listGameServices(gameId);
    if (data) {
      const ids = data.map((link) => link.service_id);
      setServiceIds(ids);
      const details: Record<number, Service> = {};
      for (const sid of ids) {
        const r = await servicesApi.getService(sid);
        if (r.data) details[sid] = r.data;
      }
      setServiceDetails(details);
    }
  }, [gameId]);

  // Results and writeups are only available to authenticated viewers; guests
  // browse the public game page without them.
  const fetchResults = useCallback(async () => {
    if (!user) return;
    const { data } = await resultsApi.listResults({ game_id: gameId });
    if (data) setResults(data.items);
  }, [gameId, user]);

  const fetchWriteups = useCallback(async () => {
    if (!user) return;
    const { data } = await writeupsApi.listWriteups({ game_id: gameId });
    if (data) setWriteups(data.items);
  }, [gameId, user]);

  const fetchAllTeams = useCallback(async () => {
    setAllTeams(await teamsApi.listAllTeams());
  }, []);

  const fetchAllServices = useCallback(async () => {
    setAllServices(await servicesApi.listAllServices());
  }, []);

  useEffect(() => {
    void fetchGame();
    void fetchGameTeams();
    void fetchServices();
    void fetchResults();
    void fetchWriteups();
    void fetchAllTeams();
    void fetchAllServices();
  }, [
    fetchGame,
    fetchGameTeams,
    fetchServices,
    fetchResults,
    fetchWriteups,
    fetchAllTeams,
    fetchAllServices,
  ]);

  // The organizer is free text; if it names a team, link to it. The team list
  // endpoint caps per_page, so resolve it via a targeted search rather than
  // scanning the (truncated) full list.
  useEffect(() => {
    const org = game?.organizer?.trim();
    if (!org) {
      setOrganizerTeam(null);
      return;
    }
    void teamsApi.listTeams({ q: org, per_page: 20 }).then((r) => {
      const match =
        r.data?.items.find((t) => t.name.toLowerCase() === org.toLowerCase()) ??
        null;
      setOrganizerTeam(match);
    });
  }, [game?.organizer]);

  const nameOf = useCallback(
    (tid: number) =>
      teamNames[tid] ??
      allTeams.find((t) => t.id === tid)?.name ??
      `${t("Team")} #${tid}`,
    [teamNames, allTeams, t],
  );

  const handleSave = async () => {
    setSaving(true);
    const { data, error: err } = await gamesApi.updateGame(gameId, {
      ...editForm,
      starts_at: datetimeLocalToRFC3339(editForm.starts_at),
      ends_at: datetimeLocalToRFC3339(editForm.ends_at),
      registration_opens_at: datetimeLocalToRFC3339(
        editForm.registration_opens_at,
      ),
      registration_closes_at: datetimeLocalToRFC3339(
        editForm.registration_closes_at,
      ),
      scoreboard_opens_at: datetimeLocalToRFC3339(editForm.scoreboard_opens_at),
      scoreboard_closes_at: datetimeLocalToRFC3339(
        editForm.scoreboard_closes_at,
      ),
    });
    setSaving(false);
    if (err) {
      setError(err);
      return;
    }
    if (data) {
      setGame(data);
      setEditing(false);
    }
  };

  const startEdit = () => {
    if (!game) return;
    setEditForm({
      name: game.name ?? undefined,
      organizer: game.organizer ?? undefined,
      starts_at: rfc3339ToDatetimeLocal(game.starts_at),
      ends_at: rfc3339ToDatetimeLocal(game.ends_at),
      avatar_url: game.avatar_url ?? undefined,
      site_url: game.site_url ?? undefined,
      ctftime_url: game.ctftime_url ?? undefined,
      registration_opens_at: rfc3339ToDatetimeLocal(game.registration_opens_at),
      registration_closes_at: rfc3339ToDatetimeLocal(
        game.registration_closes_at,
      ),
      scoreboard_opens_at: rfc3339ToDatetimeLocal(game.scoreboard_opens_at),
      scoreboard_closes_at: rfc3339ToDatetimeLocal(game.scoreboard_closes_at),
      vpn_url: game.vpn_url ?? undefined,
      vpn_config_url: game.vpn_config_url ?? undefined,
      access_instructions: game.access_instructions ?? undefined,
      access_secret: game.access_secret ?? undefined,
    });
    setEditing(true);
  };

  const handleFinalize = async () => {
    const { data, error: err } = await gamesApi.finalizeGame(gameId);
    if (err) {
      setError(err);
      return;
    }
    if (data) setGame(data);
  };

  const handleUnfinalize = async () => {
    const { data, error: err } = await gamesApi.unfinalizeGame(gameId);
    if (err) {
      setError(err);
      return;
    }
    if (data) setGame(data);
  };

  const handleExportCtf01d = async () => {
    try {
      const { data, error: err } = await gamesApi.exportCtf01d(gameId);
      if (err) {
        setError(handleApiError(err));
        return;
      }
      if (data) {
        const blob = data as unknown as Blob;
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = `ctf01d-game-${gameId}.zip`;
        a.click();
        URL.revokeObjectURL(url);
      }
    } catch (e) {
      setError(handleApiError(e));
    }
  };

  const handleAddTeam = async (e: React.FormEvent) => {
    e.preventDefault();
    const { error: err } = await gameTeamsApi.createGameTeam(addTeamForm);
    if (err) {
      setError(err);
      return;
    }
    setAddTeamForm({ game_id: gameId, team_id: 0 });
    await fetchGameTeams();
  };

  const handleRemoveTeam = async (gtId: number) => {
    const { error: err } = await gameTeamsApi.deleteGameTeam(gtId);
    if (err) {
      setError(err);
      return;
    }
    await fetchGameTeams();
  };

  const handleReorder = async () => {
    const items = gameTeams
      .sort((a, b) => a.order - b.order)
      .map((gt, i) => ({ id: gt.id, order: i + 1 }));
    const { error: err } = await gameTeamsApi.reorderGameTeams(gameId, items);
    if (err) {
      setError(err);
      return;
    }
    await fetchGameTeams();
  };

  const handleAddService = async (e: React.FormEvent) => {
    e.preventDefault();
    const sid = Number(addServiceId);
    if (!sid) return;
    const { error: err } = await gamesApi.addGameService(gameId, sid);
    if (err) {
      setError(err);
      return;
    }
    setAddServiceId("");
    await fetchServices();
  };

  const handleRemoveService = async (sid: number) => {
    const { error: err } = await gamesApi.removeGameService(gameId, sid);
    if (err) {
      setError(err);
      return;
    }
    await fetchServices();
  };

  const handleAddResult = async (e: React.FormEvent) => {
    e.preventDefault();
    const { error: err } = await resultsApi.createResult(addResultForm);
    if (err) {
      setError(err);
      return;
    }
    setAddResultForm({ game_id: gameId, team_id: 0, score: 0 });
    await fetchResults();
  };

  const handleDeleteResult = async (rid: number) => {
    const { error: err } = await resultsApi.deleteResult(rid);
    if (err) {
      setError(err);
      return;
    }
    await fetchResults();
  };

  const handleAddWriteup = async (e: React.FormEvent) => {
    e.preventDefault();
    const { error: err } = await writeupsApi.createWriteup(addWriteupForm);
    if (err) {
      setError(err);
      return;
    }
    setAddWriteupForm({
      game_id: gameId,
      team_id: 0,
      title: "",
      url: "https://",
    });
    await fetchWriteups();
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
  if (!game) return <ErrorDisplay error={error} onRetry={fetchGame} />;

  const canEdit = isPlayer;
  const canManageWriteups = isAdmin || manageableTeamIds.length > 0;
  const title = game.name ?? `${t("Game")} #${game.id}`;

  const organizerNode = game.organizer ? (
    organizerTeam ? (
      <TeamLink id={organizerTeam.id} name={organizerTeam.name} />
    ) : (
      game.organizer
    )
  ) : (
    "—"
  );

  const rosterTeamIds = new Set(gameTeams.map((gt) => gt.team_id));
  const availableTeams = allTeams.filter((t) => !rosterTeamIds.has(t.id));
  const availableServices = allServices.filter(
    (s) => !serviceIds.includes(s.id),
  );
  const rankedResults = [...results].sort(
    (a, b) => (b.score ?? 0) - (a.score ?? 0),
  );
  const writeupTeamOptions = isAdmin
    ? gameTeams.map((gt) => ({ id: gt.team_id, name: nameOf(gt.team_id) }))
    : manageableTeamIds.map((tid) => ({ id: tid, name: nameOf(tid) }));
  const hasGameLinks = Boolean(
    game.site_url || game.ctftime_url || game.vpn_url || game.vpn_config_url,
  );
  const hasAdminAccess = Boolean(
    isAdmin && (game.access_secret || game.access_instructions),
  );
  const gameTabs: Array<{ href: string; label: string; count?: number }> = [
    { href: "#overview", label: t("Overview") },
    { href: "#services", label: t("Services"), count: serviceIds.length },
    { href: "#teams", label: t("Teams"), count: gameTeams.length },
    ...(user
      ? [{ href: "#results", label: t("Results"), count: results.length }]
      : []),
    ...(user
      ? [{ href: "#writeups", label: t("Writeups"), count: writeups.length }]
      : []),
    ...(canEdit ? [{ href: "#actions", label: t("Actions") }] : []),
  ];
  const textFields: Array<{ field: keyof GameUpdate; label: string }> = [
    { field: "name", label: t("Name") },
    { field: "organizer", label: t("Organizer") },
    { field: "avatar_url", label: t("Avatar URL") },
    { field: "site_url", label: t("Site URL") },
    { field: "ctftime_url", label: t("CTFtime URL") },
    { field: "vpn_url", label: t("VPN URL") },
    { field: "vpn_config_url", label: t("VPN config URL") },
    { field: "access_instructions", label: t("Access instructions") },
    { field: "access_secret", label: t("Access secret") },
  ];
  const datetimeFields: Array<{ field: keyof GameUpdate; label: string }> = [
    { field: "starts_at", label: t("Starts At") },
    { field: "ends_at", label: t("Ends At") },
    {
      field: "registration_opens_at",
      label: t("Registration Opens At"),
    },
    {
      field: "registration_closes_at",
      label: t("Registration Closes At"),
    },
    { field: "scoreboard_opens_at", label: t("Scoreboard Opens At") },
    { field: "scoreboard_closes_at", label: t("Scoreboard Closes At") },
  ];

  return (
    <div className="page detail-page">
      <ErrorDisplay error={error} onRetry={fetchGame} />

      {!editing ? (
        <>
          <DetailHero
            kicker={`${t("Game")} #${game.id}`}
            title={title}
            avatarUrl={game.avatar_url}
            avatarText={title}
            badges={
              <>
                <CardBadge variant={game.status ?? "unknown"}>
                  {t(game.status ?? "unknown")}
                </CardBadge>
                {game.finalized && (
                  <CardBadge variant="approved">{t("finalized")}</CardBadge>
                )}
                <CardBadge variant={game.registration_status ?? "unscheduled"}>
                  {t("Registration")}{" "}
                  {t(game.registration_status ?? "unscheduled")}
                </CardBadge>
                <CardBadge variant={game.scoreboard_status ?? "closed"}>
                  {t("Scoreboard")} {t(game.scoreboard_status ?? "closed")}
                </CardBadge>
              </>
            }
            summary={[
              { label: t("Organizer"), value: organizerNode },
              {
                label: t("Starts"),
                value: <RelativeTime value={game.starts_at} />,
              },
              {
                label: t("Duration"),
                value: <Duration start={game.starts_at} end={game.ends_at} />,
              },
              ...(game.finalized
                ? [
                    {
                      label: t("Finalized"),
                      value: game.finalized_at
                        ? formatDate(game.finalized_at)
                        : t("Yes"),
                    },
                  ]
                : []),
            ]}
            actions={
              <>
                <button
                  className="btn btn-sm"
                  onClick={() => navigate("/games")}
                >
                  {t("Back")}
                </button>
                {game.site_url && (
                  <a
                    className="btn btn-sm"
                    href={safeHref(game.site_url)}
                    target="_blank"
                    rel="noreferrer"
                  >
                    {t("Site")}
                  </a>
                )}
                {game.ctftime_url && (
                  <a
                    className="btn btn-sm"
                    href={safeHref(game.ctftime_url)}
                    target="_blank"
                    rel="noreferrer"
                  >
                    CTFtime
                  </a>
                )}
                {game.vpn_url && (
                  <a
                    className="btn btn-sm"
                    href={safeHref(game.vpn_url)}
                    target="_blank"
                    rel="noreferrer"
                  >
                    VPN
                  </a>
                )}
                {canEdit && (
                  <button
                    className="btn btn-sm btn-primary"
                    onClick={startEdit}
                  >
                    {t("Edit")}
                  </button>
                )}
              </>
            }
          />

          <nav
            className="tabs game-context-tabs"
            aria-label={t("Game navigation")}
          >
            {gameTabs.map((item) => (
              <a className="tab" href={item.href} key={item.href}>
                <span>{item.label}</span>
                {typeof item.count === "number" && (
                  <span className="section-count">{item.count}</span>
                )}
              </a>
            ))}
            <Link className="tab" to={`/scoreboard?game=${game.id}`}>
              {t("Scoreboard")}
            </Link>
          </nav>

          <div className="detail-section" id="overview">
            <div className="section-head">
              <h3>{t("Game Info")}</h3>
            </div>
            <InfoGroups className="game-info-groups">
              <InfoGroup title={t("Schedule")}>
                <InfoRow label={t("Starts")}>
                  {formatDate(game.starts_at)}
                </InfoRow>
                <InfoRow label={t("Ends")}>{formatDate(game.ends_at)}</InfoRow>
                <InfoRow label={t("Duration")}>
                  <Duration start={game.starts_at} end={game.ends_at} />
                </InfoRow>
              </InfoGroup>

              <InfoGroup title={t("Registration")}>
                <InfoRow label={t("Status")}>
                  <CardBadge
                    variant={game.registration_status ?? "unscheduled"}
                  >
                    {t(game.registration_status ?? "unscheduled")}
                  </CardBadge>
                </InfoRow>
                <InfoRow label={t("Opens")}>
                  {formatDate(game.registration_opens_at)}
                </InfoRow>
                <InfoRow label={t("Closes")}>
                  {formatDate(game.registration_closes_at)}
                </InfoRow>
              </InfoGroup>

              <InfoGroup title={t("Scoreboard")}>
                <InfoRow label={t("Status")}>
                  <CardBadge variant={game.scoreboard_status ?? "closed"}>
                    {t(game.scoreboard_status ?? "closed")}
                  </CardBadge>
                </InfoRow>
                <InfoRow label={t("Opens")}>
                  {formatDate(game.scoreboard_opens_at)}
                </InfoRow>
                <InfoRow label={t("Closes")}>
                  {formatDate(game.scoreboard_closes_at)}
                </InfoRow>
              </InfoGroup>

              {hasGameLinks && (
                <InfoGroup title={t("Links")}>
                  {game.site_url && (
                    <InfoRow label={t("Site")}>
                      {renderLink(game.site_url)}
                    </InfoRow>
                  )}
                  {game.ctftime_url && (
                    <InfoRow label={t("CTFtime")}>
                      {renderLink(game.ctftime_url)}
                    </InfoRow>
                  )}
                  {game.vpn_url && (
                    <InfoRow label={t("VPN")}>
                      {renderLink(game.vpn_url)}
                    </InfoRow>
                  )}
                  {game.vpn_config_url && (
                    <InfoRow label={t("VPN config")}>
                      {renderLink(game.vpn_config_url)}
                    </InfoRow>
                  )}
                </InfoGroup>
              )}

              {hasAdminAccess && (
                <InfoGroup
                  title={t("Access (admin)")}
                  className="game-info-access"
                >
                  {game.access_secret && (
                    <InfoRow label={t("Secret")}>
                      <code>{game.access_secret}</code>
                    </InfoRow>
                  )}
                  {game.access_instructions && (
                    <InfoRow label={t("Instructions")}>
                      {game.access_instructions}
                    </InfoRow>
                  )}
                </InfoGroup>
              )}
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
          {textFields.map(({ field, label }) => (
            <div className="form-group" key={field}>
              <label>{label}</label>
              <input
                value={(editForm[field] as string) ?? ""}
                onChange={(e) =>
                  setEditForm((f) => ({ ...f, [field]: e.target.value }))
                }
              />
            </div>
          ))}
          {datetimeFields.map(({ field, label }) => (
            <div className="form-group" key={field}>
              <label>{label}</label>
              <input
                type="datetime-local"
                value={(editForm[field] as string) ?? ""}
                onChange={(e) =>
                  setEditForm((f) => ({ ...f, [field]: e.target.value }))
                }
              />
            </div>
          ))}
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

      {canEdit && (
        <div className="detail-section" id="actions">
          <div className="section-head">
            <h3>{t("Actions")}</h3>
          </div>
          <div className="action-buttons">
            {game.finalized ? (
              <ActionButton onClick={handleUnfinalize}>
                {t("Unfinalize")}
              </ActionButton>
            ) : (
              <ActionButton
                onClick={handleFinalize}
                confirm={t("Finalize this game?")}
              >
                {t("Finalize")}
              </ActionButton>
            )}
            <ActionButton onClick={handleExportCtf01d}>
              {t("Export ctf01d")}
            </ActionButton>
            <ActionButton
              onClick={() => {
                void gamesApi.deleteGame(gameId).then(() => navigate("/games"));
              }}
              variant="danger"
              confirm={t("Delete this game?")}
            >
              {t("Delete")}
            </ActionButton>
          </div>
        </div>
      )}

      <div className="detail-section" id="services">
        <div className="section-head">
          <h3>
            {t("Services")} <SectionCount n={serviceIds.length} />
          </h3>
        </div>
        {serviceIds.length > 0 ? (
          <div className="chip-list">
            {serviceIds.map((sid) => (
              <span className="entity-chip" key={sid}>
                <a href={`/services/${sid}`}>
                  {serviceDetails[sid]?.name ?? `${t("Service")} #${sid}`}
                </a>
                {canEdit && (
                  <button
                    type="button"
                    className="chip-remove"
                    title={t("Unlink service")}
                    onClick={() => void handleRemoveService(sid)}
                  >
                    ×
                  </button>
                )}
              </span>
            ))}
          </div>
        ) : (
          <p className="section-empty">{t("No services linked.")}</p>
        )}
        {canEdit && (
          <form
            onSubmit={(e) => void handleAddService(e)}
            className="inline-form"
          >
            <FilterSelect
              placeholder={t("Search services...")}
              required
              value={addServiceId ? Number(addServiceId) : null}
              onChange={(id) => setAddServiceId(id ? String(id) : "")}
              options={availableServices.map((s) => ({
                id: s.id,
                label: s.name,
                search: `${s.public_description ?? ""} ${
                  s.private_description ?? ""
                }`,
              }))}
            />
            <button type="submit" className="btn btn-sm">
              {t("Link service")}
            </button>
          </form>
        )}
      </div>

      <div className="detail-section" id="teams">
        <div className="section-head">
          <h3>
            {t("Roster")} <SectionCount n={gameTeams.length} />
          </h3>
        </div>
        {gameTeams.length > 0 ? (
          <table className="data-table">
            <thead>
              <tr>
                <th className="rank-cell">#</th>
                <th>{t("Team")}</th>
                <th>{t("IP address")}</th>
                {canEdit && <th></th>}
              </tr>
            </thead>
            <tbody>
              {gameTeams
                .sort((a, b) => a.order - b.order)
                .map((gt) => (
                  <tr key={gt.id}>
                    <td className="rank-cell">{gt.order}</td>
                    <td>
                      <TeamLink id={gt.team_id} name={nameOf(gt.team_id)} />
                    </td>
                    <td>
                      {gt.ip_address ? (
                        <code>{gt.ip_address}</code>
                      ) : (
                        <span className="muted-dash">—</span>
                      )}
                    </td>
                    {canEdit && (
                      <td className="actions-cell">
                        <ActionButton
                          onClick={() => void handleRemoveTeam(gt.id)}
                          variant="danger"
                        >
                          {t("Remove")}
                        </ActionButton>
                      </td>
                    )}
                  </tr>
                ))}
            </tbody>
          </table>
        ) : (
          <p className="section-empty">{t("No teams in roster.")}</p>
        )}
        {canEdit && (
          <form onSubmit={(e) => void handleAddTeam(e)} className="inline-form">
            <select
              value={addTeamForm.team_id || ""}
              onChange={(e) =>
                setAddTeamForm((f) => ({
                  ...f,
                  team_id: Number(e.target.value),
                }))
              }
              required
            >
              <option value="">{t("Select team...")}</option>
              {availableTeams.map((t) => (
                <option key={t.id} value={t.id}>
                  {t.name}
                </option>
              ))}
            </select>
            <input
              placeholder={t("IP address (optional)")}
              value={addTeamForm.ip_address ?? ""}
              onChange={(e) =>
                setAddTeamForm((f) => ({ ...f, ip_address: e.target.value }))
              }
            />
            <button type="submit" className="btn btn-sm">
              {t("Add team")}
            </button>
            <button
              type="button"
              className="btn btn-sm"
              onClick={() => void handleReorder()}
            >
              {t("Reorder")}
            </button>
          </form>
        )}
      </div>

      {user && (
        <>
          <div className="detail-section" id="results">
            <div className="section-head">
              <h3>
                {t("Results")} <SectionCount n={results.length} />
              </h3>
            </div>
            {rankedResults.length > 0 ? (
              <table className="data-table">
                <thead>
                  <tr>
                    <th className="rank-cell">{t("Rank")}</th>
                    <th>{t("Team")}</th>
                    <th className="numeric">{t("Score")}</th>
                    {canEdit && <th></th>}
                  </tr>
                </thead>
                <tbody>
                  {rankedResults.map((r, i) => (
                    <tr
                      key={r.id}
                      className={
                        i < 3 ? `is-podium podium-${i + 1}` : undefined
                      }
                    >
                      <td className="rank-cell">
                        {i < 3 ? (
                          <span className={`medal medal-${i + 1}`}>
                            {i + 1}
                          </span>
                        ) : (
                          i + 1
                        )}
                      </td>
                      <td>
                        <TeamLink id={r.team_id} name={nameOf(r.team_id)} />
                      </td>
                      <td className="numeric score-cell">
                        {r.score?.toLocaleString() ?? "—"}
                      </td>
                      {canEdit && (
                        <td className="actions-cell">
                          <ActionButton
                            onClick={() => void handleDeleteResult(r.id)}
                            variant="danger"
                            confirm={t("Delete this result?")}
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
              <p className="section-empty">{t("No results yet.")}</p>
            )}
            {canEdit && (
              <form
                onSubmit={(e) => void handleAddResult(e)}
                className="inline-form"
              >
                <select
                  value={addResultForm.team_id || ""}
                  onChange={(e) =>
                    setAddResultForm((f) => ({
                      ...f,
                      team_id: Number(e.target.value),
                    }))
                  }
                  required
                >
                  <option value="">{t("Select team...")}</option>
                  {gameTeams.map((gt) => (
                    <option key={gt.id} value={gt.team_id}>
                      {nameOf(gt.team_id)}
                    </option>
                  ))}
                </select>
                <input
                  type="number"
                  placeholder={t("Score")}
                  value={addResultForm.score ?? ""}
                  onChange={(e) =>
                    setAddResultForm((f) => ({
                      ...f,
                      score: Number(e.target.value),
                    }))
                  }
                  required
                />
                <button type="submit" className="btn btn-sm">
                  {t("Add result")}
                </button>
              </form>
            )}
          </div>

          <div className="detail-section" id="writeups">
            <div className="section-head">
              <h3>
                {t("Writeups")} <SectionCount n={writeups.length} />
              </h3>
            </div>
            {writeups.length > 0 ? (
              <table className="data-table">
                <thead>
                  <tr>
                    <th>{t("Team")}</th>
                    <th>{t("Title")}</th>
                    <th>{t("Link")}</th>
                    {canManageWriteups && <th></th>}
                  </tr>
                </thead>
                <tbody>
                  {writeups.map((w) => (
                    <tr key={w.id}>
                      <td>{nameOf(w.team_id)}</td>
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
                      {canManageWriteups && (
                        <td className="actions-cell">
                          {(isAdmin ||
                            manageableTeamIds.includes(w.team_id)) && (
                            <ActionButton
                              onClick={() => void handleDeleteWriteup(w.id)}
                              variant="danger"
                              confirm={t("Delete this writeup?")}
                            >
                              {t("Delete")}
                            </ActionButton>
                          )}
                        </td>
                      )}
                    </tr>
                  ))}
                </tbody>
              </table>
            ) : (
              <p className="section-empty">{t("No writeups yet.")}</p>
            )}
            {canManageWriteups && (
              <form
                onSubmit={(e) => void handleAddWriteup(e)}
                className="inline-form"
              >
                <select
                  value={addWriteupForm.team_id || ""}
                  onChange={(e) =>
                    setAddWriteupForm((f) => ({
                      ...f,
                      team_id: Number(e.target.value),
                    }))
                  }
                  required
                >
                  <option value="">{t("Select team...")}</option>
                  {writeupTeamOptions.map((t) => (
                    <option key={t.id} value={t.id}>
                      {t.name}
                    </option>
                  ))}
                </select>
                <input
                  placeholder={t("Title")}
                  value={addWriteupForm.title}
                  onChange={(e) =>
                    setAddWriteupForm((f) => ({ ...f, title: e.target.value }))
                  }
                  required
                />
                <input
                  placeholder="https://..."
                  value={addWriteupForm.url}
                  onChange={(e) =>
                    setAddWriteupForm((f) => ({ ...f, url: e.target.value }))
                  }
                  required
                />
                <button type="submit" className="btn btn-sm">
                  {t("Add writeup")}
                </button>
              </form>
            )}
          </div>
        </>
      )}
    </div>
  );
}
