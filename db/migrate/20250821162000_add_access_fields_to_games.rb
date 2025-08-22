class AddAccessFieldsToGames < ActiveRecord::Migration[8.0]
  def change
    add_column :games, :vpn_url, :string
    add_column :games, :vpn_config_url, :string
    add_column :games, :access_instructions, :text
    add_column :games, :access_secret, :string
  end
end
