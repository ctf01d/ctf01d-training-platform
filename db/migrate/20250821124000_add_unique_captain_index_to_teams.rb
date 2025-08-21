class AddUniqueCaptainIndexToTeams < ActiveRecord::Migration[8.0]
  def change
    # Заменяем обычный индекс на уникальный частичный (NULL допускается)
    remove_index :teams, column: :captain_id, name: "index_teams_on_captain_id"
    add_index :teams, :captain_id, unique: true, where: "captain_id IS NOT NULL", name: "index_teams_on_captain_id_unique"
  end
end
