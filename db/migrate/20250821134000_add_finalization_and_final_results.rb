class AddFinalizationAndFinalResults < ActiveRecord::Migration[8.0]
  def change
    add_column :games, :finalized, :boolean, default: false, null: false
    add_column :games, :finalized_at, :datetime

    create_table :final_results do |t|
      t.belongs_to :game, null: false, foreign_key: true
      t.belongs_to :team, null: false, foreign_key: true
      t.integer :score, null: false, default: 0
      t.integer :position
      t.timestamps
    end
    add_index :final_results, [ :game_id, :team_id ], unique: true
  end
end
