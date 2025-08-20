class CreateGames < ActiveRecord::Migration[8.0]
  def change
    create_table :games do |t|
      t.string :name
      t.string :organizer
      t.datetime :starts_at
      t.datetime :ends_at

      t.timestamps
    end
  end
end
