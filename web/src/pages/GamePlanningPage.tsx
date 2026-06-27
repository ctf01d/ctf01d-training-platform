import { useState, useEffect, useCallback } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import ReactMarkdown from "react-markdown";
import * as gamesApi from "../api/games";
import type { Game } from "../api/games";
import * as servicesApi from "../api/services";
import type { Service } from "../api/services";
import { CardBadge } from "../components/Card";
import { ErrorDisplay } from "../components/ErrorDisplay";
import { usePageTitle } from "../components/usePageTitle";
import { useAuth } from "../auth/AuthContext";
import { useI18n } from "../i18n/I18nContext";

type ServiceStatusOption = {
  value: string;
  label: string;
  variant: string;
};

// Planning states mirror the CyberSibir doc; variants reuse existing badge
// colors (red = early/rejected, yellow = mid, green = done/accepted).
const STATUS_OPTIONS: ServiceStatusOption[] = [
  { value: "planning", label: "Requirements", variant: "rejected" },
  { value: "design", label: "Design", variant: "rejected" },
  { value: "in_progress", label: "In progress", variant: "pending" },
  { value: "ready", label: "Ready", variant: "ok" },
  { value: "review", label: "Review", variant: "upcoming" },
  { value: "accepted", label: "Accepted", variant: "approved" },
];

function statusMeta(value: string): ServiceStatusOption {
  return (
    STATUS_OPTIONS.find((o) => o.value === value) ?? {
      value,
      label: value,
      variant: "unknown",
    }
  );
}

type Training = {
  author?: string;
  description?: string;
  vulnerabilities?: Array<{ name?: string; description?: string }>;
};

function training(service: Service): Training {
  return (service.ctf01d_training as Training | null) ?? {};
}

function servicePorts(service: Service): string {
  const ports = service.ports ?? [];
  return ports.length ? Array.from(new Set(ports)).join(", ") : "—";
}

function serviceTech(service: Service): string {
  const tech = service.tech_stack ?? [];
  return tech.length ? tech.join(", ") : "—";
}

function serviceExecutor(service: Service): string {
  return service.author ?? training(service).author ?? "—";
}

export default function GamePlanningPage() {
  const { t } = useI18n();
  const { id } = useParams<{ id: string }>();
  const gameId = Number(id);
  const navigate = useNavigate();
  const { isPlayer } = useAuth();

  const [game, setGame] = useState<Game | null>(null);
  const [links, setLinks] = useState<gamesApi.GameServiceLink[]>([]);
  const [serviceDetails, setServiceDetails] = useState<Record<number, Service>>(
    {},
  );
  const [allServices, setAllServices] = useState<Service[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<{ message?: string } | null>(null);

  const [theme, setTheme] = useState("");
  const [requirements, setRequirements] = useState("");
  const [editing, setEditing] = useState(false);
  const [saving, setSaving] = useState(false);
  const [addServiceId, setAddServiceId] = useState("");
  const [publishing, setPublishing] = useState(false);

  usePageTitle(
    game?.name
      ? t("Planning: {name}", { name: game.name })
      : t("Game planning"),
  );

  const fetchGame = useCallback(async () => {
    const { data, error: err } = await gamesApi.getGame(gameId);
    if (err) {
      setError(err);
      return;
    }
    if (data) {
      setGame(data);
      setTheme(data.theme ?? "");
      setRequirements(data.requirements ?? "");
    }
  }, [gameId]);

  const fetchServices = useCallback(async () => {
    const { data } = await gamesApi.listGameServices(gameId);
    if (data) {
      setLinks(data);
      const details: Record<number, Service> = {};
      for (const link of data) {
        const r = await servicesApi.getService(link.service_id);
        if (r.data) details[link.service_id] = r.data;
      }
      setServiceDetails(details);
    }
  }, [gameId]);

  useEffect(() => {
    setLoading(true);
    void Promise.all([fetchGame(), fetchServices()]).finally(() =>
      setLoading(false),
    );
  }, [fetchGame, fetchServices]);

  useEffect(() => {
    void servicesApi.listAllServices().then(setAllServices);
  }, []);

  const handleSaveDoc = async () => {
    setSaving(true);
    const { error: err } = await gamesApi.updateGame(gameId, {
      theme,
      requirements,
    });
    setSaving(false);
    if (err) {
      setError(err);
      return;
    }
    setEditing(false);
    await fetchGame();
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

  const handleStatusChange = async (sid: number, status: string) => {
    const { error: err } = await gamesApi.setGameServiceStatus(
      gameId,
      sid,
      status,
    );
    if (err) {
      setError(err);
      return;
    }
    await fetchServices();
  };

  const handlePublish = async () => {
    setPublishing(true);
    const { error: err } = await gamesApi.publishGame(gameId);
    setPublishing(false);
    if (err) {
      setError(err);
      return;
    }
    navigate(`/games/${gameId}`);
  };

  if (loading) return <div className="loading">{t("Loading...")}</div>;
  if (!game) return <ErrorDisplay error={error} />;

  const availableServices = allServices.filter(
    (s) => !links.some((l) => l.service_id === s.id),
  );

  return (
    <div className="page game-planning-page">
      <div className="page-header">
        <div>
          <div className="planning-eyebrow">{t("Game planning")}</div>
          <h1>{game.name ?? `${t("Game")} #${game.id}`}</h1>
        </div>
        {isPlayer && (
          <div className="action-buttons">
            {game.published ? (
              <Link className="btn" to={`/games/${gameId}`}>
                {t("Open game")}
              </Link>
            ) : (
              <button
                className="btn btn-primary"
                onClick={() => void handlePublish()}
                disabled={publishing}
              >
                {publishing ? t("Publishing...") : t("Publish to games")}
              </button>
            )}
          </div>
        )}
      </div>

      {game.published && (
        <p className="planning-published-note">
          {t("This game is already published in the games section.")}
        </p>
      )}

      <ErrorDisplay error={error} />

      <section className="detail-section">
        <div className="section-head">
          <h3>{t("Theme and general requirements")}</h3>
          {isPlayer && !editing && (
            <button className="btn" onClick={() => setEditing(true)}>
              {t("Edit")}
            </button>
          )}
        </div>

        {editing ? (
          <div className="planning-edit">
            <div className="form-group">
              <label>{t("Theme")}</label>
              <textarea
                rows={2}
                value={theme}
                onChange={(e) => setTheme(e.target.value)}
              />
            </div>
            <div className="form-group">
              <label>{t("Requirements (markdown)")}</label>
              <textarea
                className="planning-md-editor"
                rows={20}
                value={requirements}
                onChange={(e) => setRequirements(e.target.value)}
              />
            </div>
            <div className="form-actions">
              <button
                className="btn btn-primary"
                onClick={() => void handleSaveDoc()}
                disabled={saving}
              >
                {saving ? t("Saving...") : t("Save")}
              </button>
              <button
                className="btn"
                onClick={() => {
                  setEditing(false);
                  setTheme(game.theme ?? "");
                  setRequirements(game.requirements ?? "");
                }}
              >
                {t("Cancel")}
              </button>
            </div>
          </div>
        ) : (
          <div className="planning-doc">
            {theme && (
              <p className="planning-theme">
                <strong>{t("Theme")}:</strong> {theme}
              </p>
            )}
            {requirements ? (
              <div className="markdown-body">
                <ReactMarkdown>{requirements}</ReactMarkdown>
              </div>
            ) : (
              <p className="section-empty">
                {t("Requirements not filled in yet.")}
              </p>
            )}
          </div>
        )}
      </section>

      <section className="detail-section">
        <div className="section-head">
          <h3>{t("Service list")}</h3>
        </div>

        {links.length > 0 ? (
          <table className="data-table planning-services-table">
            <thead>
              <tr>
                <th>{t("Service")}</th>
                <th>{t("Assignee")}</th>
                <th>{t("Name")}</th>
                <th>{t("Port(s)")}</th>
                <th>{t("Technologies")}</th>
                <th>{t("Status")}</th>
                {isPlayer && <th></th>}
              </tr>
            </thead>
            <tbody>
              {links.map((link, idx) => {
                const svc = serviceDetails[link.service_id];
                const meta = statusMeta(link.status);
                return (
                  <tr key={link.service_id}>
                    <td>{t("Service {index}", { index: idx + 1 })}</td>
                    <td>{svc ? serviceExecutor(svc) : "—"}</td>
                    <td>
                      {svc ? (
                        <Link to={`/services/${svc.id}`}>{svc.name}</Link>
                      ) : (
                        `#${link.service_id}`
                      )}
                      {svc?.public_description && (
                        <div className="planning-service-desc">
                          {svc.public_description}
                        </div>
                      )}
                    </td>
                    <td>{svc ? servicePorts(svc) : "—"}</td>
                    <td>{svc ? serviceTech(svc) : "—"}</td>
                    <td>
                      {isPlayer ? (
                        <select
                          value={link.status}
                          onChange={(e) =>
                            void handleStatusChange(
                              link.service_id,
                              e.target.value,
                            )
                          }
                        >
                          {STATUS_OPTIONS.map((o) => (
                            <option key={o.value} value={o.value}>
                              {t(o.label)}
                            </option>
                          ))}
                        </select>
                      ) : (
                        <CardBadge variant={meta.variant}>
                          {t(meta.label)}
                        </CardBadge>
                      )}
                    </td>
                    {isPlayer && (
                      <td className="actions-cell">
                        <button
                          className="btn btn-danger btn-sm"
                          onClick={() =>
                            void handleRemoveService(link.service_id)
                          }
                        >
                          {t("Remove")}
                        </button>
                      </td>
                    )}
                  </tr>
                );
              })}
            </tbody>
          </table>
        ) : (
          <p className="section-empty">{t("No services added yet.")}</p>
        )}

        {isPlayer && (
          <form className="inline-form" onSubmit={handleAddService}>
            <select
              value={addServiceId}
              onChange={(e) => setAddServiceId(e.target.value)}
            >
              <option value="">{t("Add service...")}</option>
              {availableServices.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name}
                </option>
              ))}
            </select>
            <button className="btn" type="submit" disabled={!addServiceId}>
              {t("Add")}
            </button>
          </form>
        )}
      </section>

      <section className="detail-section">
        <div className="section-head">
          <h3>{t("Service details")}</h3>
        </div>
        {links.length === 0 && (
          <p className="section-empty">{t("No services to display.")}</p>
        )}
        {links.map((link, idx) => {
          const svc = serviceDetails[link.service_id];
          if (!svc) return null;
          const trainingInfo = training(svc);
          return (
            <div key={link.service_id} className="planning-service-card">
              <h4>
                {t("Service {index}: {name}", {
                  index: idx + 1,
                  name: svc.name,
                })}
              </h4>
              <dl className="planning-service-meta">
                <div>
                  <dt>{t("Assignee")}</dt>
                  <dd>{serviceExecutor(svc)}</dd>
                </div>
                <div>
                  <dt>{t("Port(s)")}</dt>
                  <dd>{servicePorts(svc)}</dd>
                </div>
                <div>
                  <dt>{t("Technologies")}</dt>
                  <dd>{serviceTech(svc)}</dd>
                </div>
                {svc.writeup_url && (
                  <div>
                    <dt>{t("Repository")}</dt>
                    <dd>
                      <a
                        href={svc.writeup_url}
                        target="_blank"
                        rel="noreferrer"
                      >
                        {svc.writeup_url}
                      </a>
                    </dd>
                  </div>
                )}
              </dl>
              {(trainingInfo.description ?? svc.public_description) && (
                <p className="planning-service-description">
                  {trainingInfo.description ?? svc.public_description}
                </p>
              )}
              {trainingInfo.vulnerabilities &&
                trainingInfo.vulnerabilities.length > 0 && (
                  <div className="planning-vulns">
                    <strong>{t("Vulnerabilities")}:</strong>
                    <ul>
                      {trainingInfo.vulnerabilities.map((v, i) => (
                        <li key={i}>
                          {v.name ? <strong>{v.name}</strong> : null}
                          {v.name && v.description ? " — " : null}
                          {v.description}
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
            </div>
          );
        })}
      </section>
    </div>
  );
}
