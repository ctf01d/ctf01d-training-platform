CTF01D: генерация архива из CRM

Назначение: собрать zip-архив с префиксом, содержащий `data/config.yml`, html-табло, файлы чекеров и логотипы команд.

API сервиса: `Ctf01d::ExportZip.call(game:, scoreboard:, teams:, checkers:, options:)`

Параметры
- game: { id, name, start_utc, end_utc, coffee_break_start_utc?, coffee_break_end_utc?, flag_ttl_min, basic_attack_cost, defence_cost }
- scoreboard: { port, htmlfolder, random }
- teams: массив объектов { id, name, active, ip_address, logo_rel, logo_src }
- checkers: массив объектов { id, name, enabled, script_wait, round_sleep, script_rel, files: [ { src, rel } ] }
- options: { prefix, include_html=true/false, html_source_path, include_compose=true/false, compose_project }

Замечания
- `html_source_path` по умолчанию указывает на `ctf01d/data_sample/html` из репозитория (локальная копия для шаблона табло). Можно отключить включение html через `include_html: false`.
- В `teams[].logo_rel` указывается путь относительно `<WORKDIR>` (обычно `./html/images/teams/<file>`), а `logo_src` — абсолютный путь до бинарного файла логотипа в CRM.
- Для каждого чекера создаётся папка `data/checker_<id>/` и копируются указанные файлы. `script_rel` указывать относительно этой папки.
- `docker-compose.yml` включается при `options.include_compose = true`.

Результат
- Возвращается хеш: `{ filename: '<prefix>.zip', data: <binary>, size: <bytes> }`.

Валидации (соответствуют ctf01d)
- game.id — `[a-z0-9]+`, даты UTC, `end > start`, `flag_ttl_min` в 1..25, `basic_attack_cost` в 1..500.
- scoreboard.port — 11..65535, `htmlfolder` — относительный путь (рекомендуется `./html`).
- teams — ≥1, уникальные `id` и `ip_address`, IPv4, наличие `logo_src`.
- checkers — ≥1, уникальные `id`, `script_wait >= 5`, `round_sleep >= script_wait*3`, `script_rel` задан.

Пример
```ruby
result = Ctf01d::ExportZip.call(
  game: {
    id: 'test', name: 'Test Game',
    start_utc: Time.utc(2030,1,1,10), end_utc: Time.utc(2030,1,1,20),
    flag_ttl_min: 1, basic_attack_cost: 1, defence_cost: 1.0
  },
  scoreboard: { port: 8080, htmlfolder: './html', random: false },
  teams: [
    { id: 't01', name: 'Team #1', active: true, ip_address: '10.0.1.1',
      logo_rel: './html/images/teams/team01.png', logo_src: '/abs/path/team01.png' }
  ],
  checkers: [
    { id: 'service1', name: 'Service1', enabled: true,
      script_wait: 5, round_sleep: 15, script_rel: './checker.py',
      files: [ { src: '/abs/path/checker.py', rel: 'checker.py' } ] }
  ],
  options: {
    prefix: 'ctf01d_test', include_html: true,
    html_source_path: Rails.root.join('ctf01d','data_sample','html').to_s,
    include_compose: true, compose_project: 'test'
  }
)
File.binwrite('/tmp/ctf01d_test.zip', result[:data])
```

