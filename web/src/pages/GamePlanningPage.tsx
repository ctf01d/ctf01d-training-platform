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

type ServiceStatusOption = {
  value: string;
  label: string;
  variant: string;
};

// Planning states mirror the CyberSibir doc; variants reuse existing badge
// colors (red = early/rejected, yellow = mid, green = done/accepted).
const STATUS_OPTIONS: ServiceStatusOption[] = [
  { value: "planning", label: "Планирование ТЗ", variant: "rejected" },
  { value: "design", label: "Проектирование", variant: "rejected" },
  { value: "in_progress", label: "В работе", variant: "pending" },
  { value: "ready", label: "Готово", variant: "ok" },
  { value: "review", label: "Проверка", variant: "upcoming" },
  { value: "accepted", label: "Принято", variant: "approved" },
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
  service_port?: number;
  ports?: Record<string, number> | number[];
  languages?: string[];
  tech_stack?: string[];
  author?: string;
  description?: string;
  vulnerabilities?: Array<{ name?: string; description?: string }>;
};

function training(service: Service): Training {
  return (service.ctf01d_training as Training | null) ?? {};
}

function servicePorts(service: Service): string {
  const t = training(service);
  const out: number[] = [];
  if (Array.isArray(t.ports)) out.push(...t.ports);
  else if (t.ports && typeof t.ports === "object")
    out.push(...Object.values(t.ports));
  if (t.service_port) out.push(t.service_port);
  return out.length ? Array.from(new Set(out)).join(", ") : "—";
}

function serviceTech(service: Service): string {
  const t = training(service);
  const tech = t.tech_stack ?? t.languages ?? [];
  return tech.length ? tech.join(", ") : "—";
}

function serviceExecutor(service: Service): string {
  return service.author ?? training(service).author ?? "—";
}

export default function GamePlanningPage() {
  const { id } = useParams<{ id: string }>();
  const gameId = Number(id);
  const navigate = useNavigate();
  const { isPlayer } = useAuth();

  const [game, setGame] = useState<Game | null>(null);
  const [links, setLinks] = useState<gamesApi.GameServiceLink[]>([]);
  const [serviceDetails, setServiceDetails] = useState<
    Record<number, Service>
  >({});
  const [allServices, setAllServices] = useState<Service[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<{ message?: string } | null>(null);

  const [theme, setTheme] = useState("");
  const [requirements, setRequirements] = useState("");
  const [editing, setEditing] = useState(false);
  const [saving, setSaving] = useState(false);
  const [addServiceId, setAddServiceId] = useState("");
  const [publishing, setPublishing] = useState(false);

  usePageTitle(game?.name ? `Планирование: ${game.name}` : "Планирование игры");

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

  if (loading) return <div className="loading">Loading...</div>;
  if (!game) return <ErrorDisplay error={error} />;

  const availableServices = allServices.filter(
    (s) => !links.some((l) => l.service_id === s.id),
  );

  return (
    <div className="page game-planning-page">
      <div className="page-header">
        <div>
          <div className="planning-eyebrow">Планирование игры</div>
          <h1>{game.name ?? `Game #${game.id}`}</h1>
        </div>
        {isPlayer && (
          <div className="action-buttons">
            {game.published ? (
              <Link className="btn" to={`/games/${gameId}`}>
                Открыть игру
              </Link>
            ) : (
              <button
                className="btn btn-primary"
                onClick={() => void handlePublish()}
                disabled={publishing}
              >
                {publishing ? "Переносим..." : "Перенести в раздел игры"}
              </button>
            )}
          </div>
        )}
      </div>

      {game.published && (
        <p className="planning-published-note">
          Игра уже опубликована в разделе игр.
        </p>
      )}

      <ErrorDisplay error={error} />

      <section className="detail-section">
        <div className="section-head">
          <h3>Тематика и общее ТЗ</h3>
          {isPlayer && !editing && (
            <button className="btn" onClick={() => setEditing(true)}>
              Редактировать
            </button>
          )}
        </div>

        {editing ? (
          <div className="planning-edit">
            <div className="form-group">
              <label>Тематика</label>
              <textarea
                rows={2}
                value={theme}
                onChange={(e) => setTheme(e.target.value)}
              />
            </div>
            <div className="form-group">
              <label>Требования (markdown)</label>
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
                {saving ? "Сохраняем..." : "Сохранить"}
              </button>
              <button
                className="btn"
                onClick={() => {
                  setEditing(false);
                  setTheme(game.theme ?? "");
                  setRequirements(game.requirements ?? "");
                }}
              >
                Отмена
              </button>
            </div>
          </div>
        ) : (
          <div className="planning-doc">
            {theme && (
              <p className="planning-theme">
                <strong>Тематика:</strong> {theme}
              </p>
            )}
            {requirements ? (
              <div className="markdown-body">
                <ReactMarkdown>{requirements}</ReactMarkdown>
              </div>
            ) : (
              <p className="section-empty">Требования ещё не заполнены.</p>
            )}
          </div>
        )}
      </section>

      <section className="detail-section">
        <div className="section-head">
          <h3>Список сервисов</h3>
        </div>

        {links.length > 0 ? (
          <table className="data-table planning-services-table">
            <thead>
              <tr>
                <th>Сервис</th>
                <th>Исполнитель</th>
                <th>Название</th>
                <th>Порт(ы)</th>
                <th>Технологии</th>
                <th>Состояние</th>
                {isPlayer && <th></th>}
              </tr>
            </thead>
            <tbody>
              {links.map((link, idx) => {
                const svc = serviceDetails[link.service_id];
                const meta = statusMeta(link.status);
                return (
                  <tr key={link.service_id}>
                    <td>Сервис {idx + 1}</td>
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
                              {o.label}
                            </option>
                          ))}
                        </select>
                      ) : (
                        <CardBadge variant={meta.variant}>
                          {meta.label}
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
                          Убрать
                        </button>
                      </td>
                    )}
                  </tr>
                );
              })}
            </tbody>
          </table>
        ) : (
          <p className="section-empty">Сервисы ещё не добавлены.</p>
        )}

        {isPlayer && (
          <form className="inline-form" onSubmit={handleAddService}>
            <select
              value={addServiceId}
              onChange={(e) => setAddServiceId(e.target.value)}
            >
              <option value="">Добавить сервис…</option>
              {availableServices.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name}
                </option>
              ))}
            </select>
            <button className="btn" type="submit" disabled={!addServiceId}>
              Добавить
            </button>
          </form>
        )}
      </section>

      <section className="detail-section">
        <div className="section-head">
          <h3>Детали сервисов</h3>
        </div>
        {links.length === 0 && (
          <p className="section-empty">Нет сервисов для отображения.</p>
        )}
        {links.map((link, idx) => {
          const svc = serviceDetails[link.service_id];
          if (!svc) return null;
          const t = training(svc);
          return (
            <div key={link.service_id} className="planning-service-card">
              <h4>
                Сервис {idx + 1}: {svc.name}
              </h4>
              <dl className="planning-service-meta">
                <div>
                  <dt>Исполнитель</dt>
                  <dd>{serviceExecutor(svc)}</dd>
                </div>
                <div>
                  <dt>Порт(ы)</dt>
                  <dd>{servicePorts(svc)}</dd>
                </div>
                <div>
                  <dt>Технологии</dt>
                  <dd>{serviceTech(svc)}</dd>
                </div>
                {svc.writeup_url && (
                  <div>
                    <dt>Репозиторий</dt>
                    <dd>
                      <a href={svc.writeup_url} target="_blank" rel="noreferrer">
                        {svc.writeup_url}
                      </a>
                    </dd>
                  </div>
                )}
              </dl>
              {(t.description ?? svc.public_description) && (
                <p className="planning-service-description">
                  {t.description ?? svc.public_description}
                </p>
              )}
              {t.vulnerabilities && t.vulnerabilities.length > 0 && (
                <div className="planning-vulns">
                  <strong>Уязвимости:</strong>
                  <ul>
                    {t.vulnerabilities.map((v, i) => (
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
