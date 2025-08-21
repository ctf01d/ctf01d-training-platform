class AddLinksToGames < ActiveRecord::Migration[8.0]
  def change
    add_column :games, :site_url, :string
    add_column :games, :ctftime_url, :string
  end
end

