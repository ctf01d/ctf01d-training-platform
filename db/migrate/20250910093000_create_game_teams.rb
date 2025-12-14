class CreateGameTeams < ActiveRecord::Migration[8.0]
  def change
    create_table :game_teams do |t|
      t.references :game, null: false, foreign_key: true
      t.references :team, null: false, foreign_key: true
      t.string :ip_address
      t.string :ctf01d_id
      t.jsonb :ctf01d_overrides, null: false, default: {}
      t.string :team_type
      t.integer :order, null: false, default: 0

      t.timestamps
    end

    add_index :game_teams, [ :game_id, :team_id ], unique: true
    add_index :game_teams, [ :game_id, :order, :id ]

    reversible do |dir|
      dir.up do
        say_with_time "Backfilling game_teams from results" do
          game_model = Class.new(ActiveRecord::Base) do
            self.table_name = 'games'
          end
          result_model = Class.new(ActiveRecord::Base) do
            self.table_name = 'results'
          end
          game_team_model = Class.new(ActiveRecord::Base) do
            self.table_name = 'game_teams'
          end

          game_model.find_each do |g|
            rows = result_model.where(game_id: g.id).order(score: :desc, id: :asc)
            rows.each_with_index do |r, idx|
              game_team_model.find_or_create_by!(game_id: g.id, team_id: r.team_id) do |gt|
                gt.order = idx + 1
              end
            end
          end
        end
      end
    end
  end
end
