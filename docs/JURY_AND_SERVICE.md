# Как собрать архивы для журейной системы и vuln-образа

## Структура исходного сервиса

В репозитории сервиса должны быть следующие директории и файлы:

- `service/` — содержимое для vuln-образа (внутри есть `README.md` — можно использовать как описание сервиса)
- `checker/` — скрипты для проверки (будут скопированы в образ жюри)
- `writeup/` — writeup от автора
- `exploits/` — примеры эксплойтов от автора
- `.ctf01d-service.yml` — конфиг для платформы (id сервиса, параметры для жюри)
- `LICENSE` — лицензия (желательно)
- `README.md` — общее описание для разработчиков (опционально)

**Примеры репозиториев:**
- https://github.com/sea-kg/ctf01d-service-example1-py
- https://github.com/sea-kg/ctf01d-service-example2-php

---

## Сборка vuln-образа

- Директорию `service` копировать с переименованием в `%id-of-service%`, где id берётся из `.ctf01d-service.yml` (секция `checker-config-*`/`id`).

## Сборка жюрейного образа

- Директорию `checker` копировать с переименованием в `data_game/checker_%id-of-service%` (id из `.ctf01d-service.yml`).

---

## Пример docker-compose для жюрейной системы

```yaml
version: '3'

services:
  ctf01d-jury:
    build: .
    container_name: ctf01d_jury_game1
    volumes:
      - "./data_game:/usr/share/ctf01d"
    environment:
      CTF01D_WORKDIR: "/usr/share/ctf01d"
    ports:
      - "8080:8080"
    restart: always
    networks:
      - ctf01d_net

networks:
  ctf01d_net:
    driver: bridge
```

---

## Пример Dockerfile для жюрейного образа

- В секции `install-checker-requirements-*` из `.ctf01d-service.yml` добавляйте нужные команды.

```dockerfile
FROM sea5kg/ctf01d:v0.5.2

# Пример для python checker
# checker_ctf01d-service-example2-php
# copied from https://github.com/sea-kg/ctf01d-service-example2-php/blob/main/.ctf01d-service.yml

RUN apt-get -y update && \
    apt install -y python3 python3-pip python3-requests
```

---

## Пример конфига для жюрейной системы (`data_game/config.yml`)

```yaml
# Пример конфига для ctf01d
# use 2 spaces for tab

game:
  id: "game" # uniq gameid must be regexp [a-z0-9]+
  name: "Game1" # visible game name in scoreboard
  start: "2023-11-12 16:00:00" # start time of game (UTC)
  end: "2030-11-12 22:00:00" # end time of game (UTC)
  coffee_break_start: "2023-11-12 20:00:00" # start time of game coffee break (UTC), but it will be ignored if period more (or less) then start and end
  coffee_break_end: "2023-11-12 21:00:00" # end time of game coffee break (UTC), but it will be ignored if period more (or less) then start and end
  flag_timelive_in_min: 1 # you can change flag time live (in minutes)
  basic_costs_stolen_flag_in_points: 1 # basic costs stolen (attack) flag in points for adaptive scoreboard
  cost_defence_flag_in_points: 1.0 # cost defences flag in points

scoreboard:
  port: 8080 # http port for scoreboard
  htmlfolder: "./html" # web page for scoreboard see index-template.html
  random: no # If yes - will be random values in scoreboard

checkers:
  - id: "example_service1_py" # work directory will be checker_example_service1_py # copied from https://github.com/sea-kg/ctf01d-service-example1-py/blob/main/.ctf01d-service.yml
    service_name: "Service1 Py"
    enabled: yes
    script_path: "./checker.py"
    script_wait_in_sec: 5 # max time for running script
    time_sleep_between_run_scripts_in_sec: 15
  - id: "example_service2_php" # work directory will be checker_example_service2_php # copied from https://github.com/sea-kg/ctf01d-service-example2-php/blob/main/.ctf01d-service.yml
    service_name: "Service2 PHP"
    enabled: yes
    script_path: "./checker.py"
    script_wait_in_sec: 5 # max time for running script
    time_sleep_between_run_scripts_in_sec: 15

teams:
  - id: "t01" # must be uniq
    name: "Team #1"
    active: yes
    logo: "./html/images/teams/team01.png"
    ip_address: "127.0.1.1" # address to vulnserver
  - id: "t02" # must be uniq
    name: "Team #2"
    active: yes
    logo: "./html/images/teams/team02.png"
    ip_address: "127.0.2.1" # address to vulnserver
```

---

## Копирование checker-скриптов

Для каждого сервиса:
- Скопируйте содержимое директории `checker` из репозитория сервиса в `data_game/checker_%id-of-service%` (id из `.ctf01d-service.yml`).

**Пример:**
- `data_game/checker_example_service1_py` ← содержимое https://github.com/sea-kg/ctf01d-service-example1-py/tree/main/checker
- `data_game/checker_example_service2_php` ← содержимое https://github.com/sea-kg/ctf01d-service-example2-php/tree/main/checker
