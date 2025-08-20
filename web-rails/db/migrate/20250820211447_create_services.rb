class CreateServices < ActiveRecord::Migration[8.0]
  def change
    create_table :services do |t|
      t.string :name, null: false
      t.text :public_description
      t.text :private_description
      t.string :author
      t.string :copyright
      t.string :avatar_url
      t.boolean :public, null: false, default: true

      t.timestamps
    end
    add_index :services, :name, unique: true
  end
end
