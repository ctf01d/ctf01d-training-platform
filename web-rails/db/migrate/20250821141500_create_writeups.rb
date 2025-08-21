class CreateWriteups < ActiveRecord::Migration[8.0]
  def change
    create_table :writeups do |t|
      t.belongs_to :game, null: false, foreign_key: true
      t.belongs_to :team, null: false, foreign_key: true
      t.string :title, null: false
      t.string :url, null: false
      t.timestamps
    end
    add_index :writeups, [:game_id, :team_id, :title], unique: true
  end
end

