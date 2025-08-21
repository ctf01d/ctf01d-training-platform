class CreateUsers < ActiveRecord::Migration[8.0]
  def change
    create_table :users do |t|
      t.string :user_name, null: false
      t.string :display_name, null: false
      t.string :role, null: false, default: 'guest'
      t.integer :rating, null: false, default: 0
      t.string :avatar_url
      t.string :password_digest

      t.timestamps
    end
    add_index :users, :user_name, unique: true
  end
end
