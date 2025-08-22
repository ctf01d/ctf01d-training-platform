class AddUniqueIndexResultsGameTeam < ActiveRecord::Migration[8.0]
  def change
    add_index :results, [ :game_id, :team_id ], unique: true
  end
end
