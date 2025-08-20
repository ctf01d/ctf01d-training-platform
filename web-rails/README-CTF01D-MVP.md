CTF01D Rails MVP

Quick Rails prototype to demo core entities and flows before porting to Go. Uses PostgreSQL and default ERB views.

Features
- Public services list (only public items for guests)
- Admin CRUD for Users, Teams, Services, Games, Results, Universities
- Session-based auth (login/logout), seeded admin user

Prerequisites
- Ruby 3.4+
- PostgreSQL (local or Docker)

Run PostgreSQL with Docker
docker run -d --name ctf01d-postgres -e POSTGRES_DB=web_rails_development -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=postgres -p 5432:5432 postgres

Setup
cd web-rails
~/.local/share/gem/ruby/3.4.0/bin/bundle install
bin/rails db:create db:migrate db:seed

Start server
bin/rails s

Login
- URL: http://localhost:3000
- Admin: admin / admin

Notes
- Root points to `services#index` (public catalog)
- Non-admins can view; only admins can create/update/delete
- Models: User, Team, TeamMembership, Service, Game, Result, University

