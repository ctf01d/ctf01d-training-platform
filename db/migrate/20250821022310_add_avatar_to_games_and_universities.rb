class AddAvatarToGamesAndUniversities < ActiveRecord::Migration[8.0]
  def change
    add_column :games, :avatar_url, :string
    add_column :universities, :avatar_url, :string
  end
end

