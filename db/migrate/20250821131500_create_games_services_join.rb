class CreateGamesServicesJoin < ActiveRecord::Migration[8.0]
  def change
    create_table :games_services, id: false do |t|
      t.belongs_to :game, null: false, foreign_key: true, index: false
      t.belongs_to :service, null: false, foreign_key: true, index: false
    end

    add_index :games_services, [ :game_id, :service_id ], unique: true
    add_index :games_services, [ :service_id, :game_id ]
  end
end
