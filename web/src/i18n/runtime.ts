export type Language = "en" | "ru";

export type TranslationParams = Record<string, string | number>;

type TranslationValue = string | ((params: TranslationParams) => string);

const DEFAULT_LANGUAGE: Language = "en";
const STORAGE_KEY = "ui_language";

let currentLanguage: Language = DEFAULT_LANGUAGE;

const ruCatalog: Record<string, TranslationValue> = {
  Games: "Игры",
  Teams: "Команды",
  Scoreboard: "Таблица результатов",
  Results: "Результаты",
  Writeups: "Райтапы",
  Services: "Сервисы",
  Universities: "Университеты",
  Users: "Пользователи",
  Admin: "Админ",
  Profile: "Профиль",
  Login: "Вход",
  Logout: "Выйти",
  "Loading...": "Загрузка...",
  Refresh: "Обновить",
  "Refreshing...": "Обновление...",
  Cancel: "Отмена",
  Back: "Назад",
  Create: "Создать",
  Add: "Добавить",
  Save: "Сохранить",
  Edit: "Изменить",
  Delete: "Удалить",
  Remove: "Убрать",
  "Sign In": "Войти",
  "Signing in...": "Вход...",
  "Login failed": "Не удалось войти",
  Username: "Логин",
  Password: "Пароль",
  "New Password": "Новый пароль",
  "Confirm Password": "Подтвердите пароль",
  "Change password": "Сменить пароль",
  "Updating...": "Обновление...",
  "Display Name": "Отображаемое имя",
  About: "О себе",
  Telegram: "Telegram",
  GitHub: "GitHub",
  Email: "Email",
  Language: "Язык",
  English: "Английский",
  Russian: "Русский",
  Saving: "Сохранение",
  "Saving...": "Сохранение...",
  "Save profile": "Сохранить профиль",
  "Profile updated successfully.": "Профиль успешно обновлён.",
  "Password updated successfully.": "Пароль успешно обновлён.",
  "Avatar updated successfully.": "Аватар успешно обновлён.",
  "Passwords do not match.": "Пароли не совпадают.",
  "Upload avatar": "Загрузить аватар",
  "Uploading...": "Загрузка...",
  Account: "Аккаунт",
  Access: "Доступ",
  Role: "Роль",
  Action: "Действие",
  Link: "Ссылка",
  Logo: "Логотип",
  Rating: "Рейтинг",
  Status: "Статус",
  Active: "Активен",
  Blocked: "Заблокирован",
  Yes: "Да",
  No: "Нет",
  "Last IP": "Последний IP",
  "Last login": "Последний вход",
  "Active sessions": "Активные сессии",
  "No active sessions.": "Активных сессий нет.",
  current: "текущая",
  "IP address": "IP-адрес",
  Client: "Клиент",
  "Last seen": "Последняя активность",
  Started: "Начало",
  Expires: "Истекает",
  Revoke: "Отозвать",
  "Revoke this session?": "Отозвать эту сессию?",
  "Delete this result?": "Удалить этот результат?",
  Retry: "Повторить",
  Prev: "Назад",
  Next: "Вперёд",
  Page: "Страница",
  of: "из",
  items: "элем.",
  Actions: "Действия",
  Clear: "Очистить",
  Filter: "Фильтр",
  Meta: "Метаданные",
  Overview: "Обзор",
  Check: "Проверка",
  Visibility: "Видимость",
  Checked: "Проверено",
  Sources: "Источники",
  Theme: "Тема",
  Requirements: "Требования",
  Design: "Дизайн",
  "In progress": "В работе",
  Ready: "Готово",
  Review: "Ревью",
  Accepted: "Принято",
  "No matches": "Нет совпадений",
  "No data": "Нет данных",
  "No users found": "Пользователи не найдены",
  "No teams found": "Команды не найдены",
  "No universities found": "Университеты не найдены",
  "No results found": "Результаты не найдены",
  "No services found": "Сервисы не найдены",
  "No games found": "Игры не найдены",
  "No scoreboard entries": "Записей в таблице результатов нет",
  "No entries for this game": "Для этой игры записей нет",
  "An error occurred": "Произошла ошибка",
  "embedded image": "встроенное изображение",
  "request validation failed": "Ошибка валидации запроса",
  "internal server error": "Внутренняя ошибка сервера",
  "not authenticated": "Не аутентифицирован",
  "invalid JSON": "Некорректный JSON",
  "not found": "Не найдено",
  conflict: "Конфликт",
  "must be at least 6 characters": "Должно быть не короче 6 символов",
  "must not be empty": "Не должно быть пустым",
  "must be one of: en, ru": "Допустимые значения: en, ru",
  "must match ^[a-zA-Z0-9_]+$ and be non-empty":
    "Должно соответствовать ^[a-zA-Z0-9_]+$ и быть непустым",
  Guest: "Гость",
  Player: "Игрок",
  User: "Пользователь",
  Team: "Команда",
  University: "Университет",
  Game: "Игра",
  Service: "Сервис",
  "Search users...": "Поиск пользователей...",
  "Create User": "Создать пользователя",
  "Avatar URL": "URL аватара",
  "Creating...": "Создание...",
  blocked: "заблокирован",
  "Search teams...": "Поиск команд...",
  "Create Team": "Создать команду",
  Description: "Описание",
  Website: "Сайт",
  "University ID": "ID университета",
  Members: "Участники",
  "Search universities...": "Поиск университетов...",
  "Create University": "Создать университет",
  "University Info": "Информация об университете",
  Site: "Сайт",
  "Site URL": "URL сайта",
  Added: "Добавлено",
  Updated: "Обновлено",
  "Filter by Game ID": "Фильтр по ID игры",
  "Filter by Team ID": "Фильтр по ID команды",
  "Create Result": "Создать результат",
  "Game ID": "ID игры",
  "Team ID": "ID команды",
  Score: "Счёт",
  Created: "Создано",
  "Delete this service?": "Удалить этот сервис?",
  "Delete this team?": "Удалить эту команду?",
  "Delete this university?": "Удалить этот университет?",
  "Delete this game?": "Удалить эту игру?",
  "Delete this writeup?": "Удалить этот райтап?",
  "Remove this member?": "Удалить этого участника?",
  Global: "Общая",
  "By Game": "По игре",
  "Global Scoreboard": "Общая таблица результатов",
  Position: "Место",
  "Total Score": "Общий счёт",
  "Game Scoreboard": "Таблица результатов игры",
  "Select a game...": "Выберите игру...",
  "Status:": "Статус:",
  "Select a game to view its scoreboard":
    "Выберите игру, чтобы посмотреть её таблицу результатов",
  "Scoreboard scope": "Область таблицы результатов",
  All: "Все",
  Public: "Публичный",
  Private: "Приватный",
  "Search services...": "Поиск сервисов...",
  "Create Service": "Создать сервис",
  "Service Info": "Информация о сервисе",
  "Close Import": "Закрыть импорт",
  "Import Service": "Импортировать сервис",
  "Name *": "Название *",
  Author: "Автор",
  "Public Description": "Публичное описание",
  "Private Description": "Приватное описание",
  Copyright: "Копирайт",
  "Writeup URL": "URL райтапа",
  Writeup: "Райтап",
  CTFtime: "CTFtime",
  VPN: "VPN",
  Exploits: "Эксплойты",
  "Exploits URL": "URL эксплойтов",
  "Service Archive URL": "URL архива сервиса",
  "Checker Archive URL": "URL архива чекера",
  "Service URL": "URL сервиса",
  "Checker URL": "URL чекера",
  Ports: "Порты",
  "Tech stack": "Технологии",
  "Last check": "Последняя проверка",
  ZIP: "ZIP",
  "Import steps": "Шаги импорта",
  Source: "Источник",
  Validate: "Проверить",
  Import: "Импорт",
  "Repo URL *": "URL репозитория *",
  Ref: "Ref",
  Subdirectory: "Подкаталог",
  "ZIP Archive *": "ZIP-архив *",
  "Validating...": "Проверка...",
  "Importing...": "Импорт...",
  "Service ID": "ID сервиса",
  "Expected repo": "Ожидаемый репозиторий",
  "Service dir": "Каталог сервиса",
  "Checker dir": "Каталог чекера",
  public: "публичный",
  private: "приватный",
  Warning: "Предупреждение",
  Error: "Ошибка",
  "Search games...": "Поиск игр...",
  "Create Game": "Создать игру",
  Name: "Название",
  "Organizer Team": "Команда-организатор",
  "Select an existing team": "Выберите существующую команду",
  Schedule: "Расписание",
  "Exact date & time": "Точная дата и время",
  "Year only": "Только год",
  Year: "Год",
  "Plan Game": "Спланировать игру",
  "Create a draft with requirements and open planning":
    "Создать черновик с требованиями и открыть планирование",
  "In planning": "В планировании",
  Published: "Опубликовано",
  planning: "планирование",
  finalized: "финализирована",
  Organizer: "Организатор",
  Finalized: "Финализировано",
  Date: "Дата",
  Duration: "Длительность",
  Registration: "Регистрация",
  Starts: "Начало",
  Ends: "Конец",
  Opens: "Открывается",
  Closes: "Закрывается",
  Links: "Ссылки",
  "VPN config": "Конфигурация VPN",
  "Game Info": "Информация об игре",
  "Game navigation": "Навигация по игре",
  Roster: "Состав",
  Rank: "Место",
  Title: "Заголовок",
  unscheduled: "не запланировано",
  closed: "закрыто",
  upcoming: "скоро",
  open: "открыто",
  active: "активно",
  ready: "готово",
  queued: "в очереди",
  failed: "ошибка",
  unknown: "неизвестно",
  pending: "ожидает",
  approved: "подтверждено",
  rejected: "отклонено",
  owner: "владелец",
  captain: "капитан",
  vice_captain: "вице-капитан",
  created: "создано",
  join_request: "заявка на вступление",
  invite: "приглашение",
  set_role: "смена роли",
  accepted: "принято",
  declined: "отклонено",
  OK: "OK",
  Member: "Участник",
  Captain: "Капитан",
  Events: "События",
  "Team Info": "Информация о команде",
  "Role change": "Изменение роли",
  "Status change": "Изменение статуса",
  "Danger Zone": "Опасная зона",
  "Delete user": "Удалить пользователя",
  "Deleting...": "Удаление...",
  "Confirm delete": "Подтвердить удаление",
  "Your password": "Ваш пароль",
  Block: "Заблокировать",
  Unblock: "Разблокировать",
  "You cannot change your own role.": "Нельзя менять свою собственную роль.",
  "You cannot delete your own account.": "Нельзя удалить собственный аккаунт.",
  "This permanently deletes @{username} and all references to them. Confirm with your admin password.":
    "Это навсегда удалит @{username} и все связанные с ним данные. Подтвердите действие паролем администратора.",
  "No teams from this university.": "От этого университета ещё нет команд.",
  "Make Private": "Сделать приватным",
  "Make Public": "Сделать публичным",
  "Check Checker": "Проверить чекер",
  "Re-download Archives": "Перекачать архивы",
  Archives: "Архивы",
  "Service archive": "Архив сервиса",
  "Checker archive": "Архив чекера",
  "Service Archive": "Архив сервиса",
  "Checker Archive": "Архив чекера",
  "Upload Archives": "Загрузить архивы",
  present: "есть",
  none: "нет",
  Size: "Размер",
  "Not uploaded": "Не загружен",
  Download: "Скачать",
  "e.g. 8080, 9000": "например, 8080, 9000",
  "e.g. Python, PostgreSQL, nginx": "например, Python, PostgreSQL, nginx",
  "Requesting...": "Отправка...",
  "Request to Join": "Подать заявку",
  Approve: "Одобрить",
  Reject: "Отклонить",
  Accept: "Принять",
  Decline: "Отклонить",
  "No members.": "Участников нет.",
  "Invite user...": "Пригласить пользователя...",
  "Inviting...": "Приглашение...",
  Invite: "Пригласить",
  "{approved} approved · {total} total":
    "{approved} подтверждено · {total} всего",
  "Placed {rank} of {total} teams": "Заняла место {rank} из {total} команд",
  "No games played yet.": "Команда ещё не играла.",
  "Open ↗": "Открыть ↗",
  "No writeups.": "Райтапов нет.",
  "No events.": "Событий нет.",
  "Planning: {name}": "Планирование: {name}",
  "Game planning": "Планирование игры",
  "Open game": "Открыть игру",
  "Publishing...": "Публикация...",
  "Publish to games": "Опубликовать в играх",
  "This game is already published in the games section.":
    "Эта игра уже опубликована в разделе игр.",
  "Theme and general requirements": "Тема и общие требования",
  "Requirements (markdown)": "Требования (markdown)",
  "Requirements not filled in yet.": "Требования пока не заполнены.",
  "Service list": "Список сервисов",
  Assignee: "Ответственный",
  "Port(s)": "Порт(ы)",
  Technologies: "Технологии",
  "No services added yet.": "Сервисы ещё не добавлены.",
  "Add service...": "Добавить сервис...",
  "Service details": "Детали сервисов",
  "No services to display.": "Нет сервисов для отображения.",
  Repository: "Репозиторий",
  Vulnerabilities: "Уязвимости",
  "Service {index}": "Сервис {index}",
  "Service {index}: {name}": "Сервис {index}: {name}",
  "Access (admin)": "Доступ (админ)",
  Secret: "Секрет",
  Instructions: "Инструкции",
  "Starts At": "Время начала",
  "Ends At": "Время окончания",
  "Registration Opens At": "Регистрация открывается",
  "Registration Closes At": "Регистрация закрывается",
  "Scoreboard Opens At": "Табло открывается",
  "Scoreboard Closes At": "Табло закрывается",
  "CTFtime URL": "URL CTFtime",
  "VPN URL": "URL VPN",
  "VPN config URL": "URL конфигурации VPN",
  "Access instructions": "Инструкции по доступу",
  "Access secret": "Секрет доступа",
  Unfinalize: "Снять финализацию",
  "Finalize this game?": "Финализировать эту игру?",
  Finalize: "Финализировать",
  "Export ctf01d": "Экспортировать ctf01d",
  "Unlink service": "Отвязать сервис",
  "No services linked.": "Сервисы не привязаны.",
  "Link service": "Привязать сервис",
  "No teams in roster.": "В составе пока нет команд.",
  "Select team...": "Выберите команду...",
  "IP address (optional)": "IP-адрес (необязательно)",
  "Add team": "Добавить команду",
  Reorder: "Переупорядочить",
  "No results yet.": "Результатов пока нет.",
  "Add result": "Добавить результат",
  "No writeups yet.": "Райтапов пока нет.",
  "Add writeup": "Добавить райтап",
  "Account menu": "Меню аккаунта",
  "Primary navigation": "Основная навигация",
  "You do not have permission to access this resource.":
    "У вас нет прав для доступа к этому ресурсу.",
  "Something went wrong": "Что-то пошло не так",
  "Setting a new password takes effect immediately. It does not affect the rest of the profile.":
    "Новый пароль начинает действовать сразу и не затрагивает остальные поля профиля.",
  "just now": "только что",
  "less than a minute ago": "меньше минуты назад",
  "in less than a minute": "меньше чем через минуту",
  "year ago_one": "{count} год назад",
  "year ago_few": "{count} года назад",
  "year ago_many": "{count} лет назад",
  "in year_one": "через {count} год",
  "in year_few": "через {count} года",
  "in year_many": "через {count} лет",
  "month ago_one": "{count} месяц назад",
  "month ago_few": "{count} месяца назад",
  "month ago_many": "{count} месяцев назад",
  "in month_one": "через {count} месяц",
  "in month_few": "через {count} месяца",
  "in month_many": "через {count} месяцев",
  "week ago_one": "{count} неделю назад",
  "week ago_few": "{count} недели назад",
  "week ago_many": "{count} недель назад",
  "in week_one": "через {count} неделю",
  "in week_few": "через {count} недели",
  "in week_many": "через {count} недель",
  "day ago_one": "{count} день назад",
  "day ago_few": "{count} дня назад",
  "day ago_many": "{count} дней назад",
  "in day_one": "через {count} день",
  "in day_few": "через {count} дня",
  "in day_many": "через {count} дней",
  "hour ago_one": "{count} час назад",
  "hour ago_few": "{count} часа назад",
  "hour ago_many": "{count} часов назад",
  "in hour_one": "через {count} час",
  "in hour_few": "через {count} часа",
  "in hour_many": "через {count} часов",
  "minute ago_one": "{count} минуту назад",
  "minute ago_few": "{count} минуты назад",
  "minute ago_many": "{count} минут назад",
  "in minute_one": "через {count} минуту",
  "in minute_few": "через {count} минуты",
  "in minute_many": "через {count} минут",
  "second ago_one": "{count} секунду назад",
  "second ago_few": "{count} секунды назад",
  "second ago_many": "{count} секунд назад",
  "in second_one": "через {count} секунду",
  "in second_few": "через {count} секунды",
  "in second_many": "через {count} секунд",
};

export function normalizeLanguage(value?: string | null): Language {
  const language = value?.trim().toLowerCase();
  if (language?.startsWith("ru")) return "ru";
  return "en";
}

export function detectPreferredLanguage(): Language {
  if (typeof window !== "undefined") {
    const stored = window.localStorage.getItem(STORAGE_KEY);
    if (stored) return normalizeLanguage(stored);
    return normalizeLanguage(window.navigator.language);
  }
  return DEFAULT_LANGUAGE;
}

export function localeForLanguage(language: Language): string {
  return language === "ru" ? "ru-RU" : "en-US";
}

export function getCurrentLanguage(): Language {
  return currentLanguage;
}

export function setCurrentLanguage(language: Language) {
  currentLanguage = language;
  if (typeof document !== "undefined") {
    document.documentElement.lang = language;
  }
  if (typeof window !== "undefined") {
    window.localStorage.setItem(STORAGE_KEY, language);
  }
}

function interpolate(template: string, params?: TranslationParams): string {
  if (!params) return template;
  return template.replace(/\{(\w+)\}/g, (_match: string, key: string) => {
    const value = params[key];
    return value === undefined ? `{${key}}` : String(value);
  });
}

function pluralSuffix(count: number): "one" | "few" | "many" {
  const mod10 = count % 10;
  const mod100 = count % 100;
  if (mod10 === 1 && mod100 !== 11) return "one";
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14)) return "few";
  return "many";
}

export function translateLiteral(
  language: Language,
  source: string,
  params?: TranslationParams,
): string {
  if (language === "en") return interpolate(source, params);
  const value = ruCatalog[source];
  if (typeof value === "function") return value(params ?? {});
  return interpolate(value ?? source, params);
}

export function translateCurrent(
  source: string,
  params?: TranslationParams,
): string {
  return translateLiteral(currentLanguage, source, params);
}

export function roleLiteral(role?: string | null): string {
  switch (role) {
    case "guest":
      return "Guest";
    case "player":
      return "Player";
    case "admin":
      return "Admin";
    default:
      return role ?? "";
  }
}

export function languageLiteral(language?: string | null): string {
  return normalizeLanguage(language) === "ru" ? "Russian" : "English";
}

export function formatRelativeLiteral(
  count: number,
  unit: string,
  future: boolean,
): string {
  if (currentLanguage !== "ru") {
    const label = `${count} ${unit}${count === 1 ? "" : "s"}`;
    return future ? `in ${label}` : `${label} ago`;
  }
  const suffix = pluralSuffix(count);
  const key = future ? `in ${unit}_${suffix}` : `${unit} ago_${suffix}`;
  return translateCurrent(key, { count });
}
