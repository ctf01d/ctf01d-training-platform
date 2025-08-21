# This file is auto-generated from the current state of the database. Instead
# of editing this file, please use the migrations feature of Active Record to
# incrementally modify your database, and then regenerate this schema definition.
#
# This file is the source Rails uses to define your schema when running `bin/rails
# db:schema:load`. When creating a new database, `bin/rails db:schema:load` tends to
# be faster and is potentially less error prone than running all of your
# migrations from scratch. Old migrations may fail to apply correctly if those
# migrations use external dependencies or application code.
#
# It's strongly recommended that you check this file into your version control system.

ActiveRecord::Schema[8.0].define(version: 2025_08_21_162000) do
  # These are extensions that must be enabled in order to support this database
  enable_extension "pg_catalog.plpgsql"

  create_table "final_results", force: :cascade do |t|
    t.bigint "game_id", null: false
    t.bigint "team_id", null: false
    t.integer "score", default: 0, null: false
    t.integer "position"
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["game_id", "team_id"], name: "index_final_results_on_game_id_and_team_id", unique: true
    t.index ["game_id"], name: "index_final_results_on_game_id"
    t.index ["team_id"], name: "index_final_results_on_team_id"
  end

  create_table "games", force: :cascade do |t|
    t.string "name"
    t.string "organizer"
    t.datetime "starts_at"
    t.datetime "ends_at"
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.string "avatar_url"
    t.string "site_url"
    t.string "ctftime_url"
    t.boolean "finalized", default: false, null: false
    t.datetime "finalized_at"
    t.datetime "registration_opens_at"
    t.datetime "registration_closes_at"
    t.datetime "scoreboard_opens_at"
    t.datetime "scoreboard_closes_at"
    t.string "vpn_url"
    t.string "vpn_config_url"
    t.text "access_instructions"
    t.string "access_secret"
  end

  create_table "games_services", id: false, force: :cascade do |t|
    t.bigint "game_id", null: false
    t.bigint "service_id", null: false
    t.index ["game_id", "service_id"], name: "index_games_services_on_game_id_and_service_id", unique: true
    t.index ["service_id", "game_id"], name: "index_games_services_on_service_id_and_game_id"
  end

  create_table "results", force: :cascade do |t|
    t.bigint "game_id", null: false
    t.bigint "team_id", null: false
    t.integer "score"
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["game_id", "team_id"], name: "index_results_on_game_id_and_team_id", unique: true
    t.index ["game_id"], name: "index_results_on_game_id"
    t.index ["team_id"], name: "index_results_on_team_id"
  end

  create_table "services", force: :cascade do |t|
    t.string "name", null: false
    t.text "public_description"
    t.text "private_description"
    t.string "author"
    t.string "copyright"
    t.string "avatar_url"
    t.boolean "public", default: true, null: false
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.string "service_archive_url"
    t.string "checker_archive_url"
    t.string "writeup_url"
    t.string "exploits_url"
    t.string "check_status", default: "unknown", null: false
    t.datetime "checked_at"
    t.index ["name"], name: "index_services_on_name", unique: true
  end

  create_table "team_membership_events", force: :cascade do |t|
    t.bigint "team_id", null: false
    t.bigint "user_id", null: false
    t.integer "actor_id"
    t.string "action", null: false
    t.string "from_role"
    t.string "to_role"
    t.string "from_status"
    t.string "to_status"
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["actor_id"], name: "index_team_membership_events_on_actor_id"
    t.index ["team_id", "created_at"], name: "index_team_membership_events_on_team_id_and_created_at"
    t.index ["team_id"], name: "index_team_membership_events_on_team_id"
    t.index ["user_id"], name: "index_team_membership_events_on_user_id"
  end

  create_table "team_memberships", force: :cascade do |t|
    t.bigint "team_id", null: false
    t.bigint "user_id", null: false
    t.string "role"
    t.string "status"
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["team_id"], name: "index_team_memberships_on_team_id"
    t.index ["user_id"], name: "index_team_memberships_on_user_id"
  end

  create_table "teams", force: :cascade do |t|
    t.string "name", null: false
    t.text "description"
    t.string "website"
    t.string "avatar_url"
    t.integer "captain_id"
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.bigint "university_id"
    t.index ["captain_id"], name: "index_teams_on_captain_id_unique", unique: true, where: "(captain_id IS NOT NULL)"
    t.index ["university_id"], name: "index_teams_on_university_id"
  end

  create_table "universities", force: :cascade do |t|
    t.string "name"
    t.string "site_url"
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.string "avatar_url"
  end

  create_table "users", force: :cascade do |t|
    t.string "user_name", null: false
    t.string "display_name", null: false
    t.string "role", default: "guest", null: false
    t.integer "rating", default: 0, null: false
    t.string "avatar_url"
    t.string "password_digest"
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["user_name"], name: "index_users_on_user_name", unique: true
  end

  create_table "writeups", force: :cascade do |t|
    t.bigint "game_id", null: false
    t.bigint "team_id", null: false
    t.string "title", null: false
    t.string "url", null: false
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["game_id", "team_id", "title"], name: "index_writeups_on_game_id_and_team_id_and_title", unique: true
    t.index ["game_id"], name: "index_writeups_on_game_id"
    t.index ["team_id"], name: "index_writeups_on_team_id"
  end

  add_foreign_key "final_results", "games"
  add_foreign_key "final_results", "teams"
  add_foreign_key "games_services", "games"
  add_foreign_key "games_services", "services"
  add_foreign_key "results", "games"
  add_foreign_key "results", "teams"
  add_foreign_key "team_membership_events", "teams"
  add_foreign_key "team_membership_events", "users"
  add_foreign_key "team_memberships", "teams"
  add_foreign_key "team_memberships", "users"
  add_foreign_key "teams", "universities"
  add_foreign_key "writeups", "games"
  add_foreign_key "writeups", "teams"
end
