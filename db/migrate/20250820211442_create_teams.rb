class CreateTeams < ActiveRecord::Migration[8.0]
  def change
    create_table :teams do |t|
      t.string :name, null: false
      t.string :university
      t.text :description
      t.string :website
      t.string :avatar_url
      t.integer :captain_id

      t.timestamps
    end
    add_index :teams, :captain_id
  end
end
