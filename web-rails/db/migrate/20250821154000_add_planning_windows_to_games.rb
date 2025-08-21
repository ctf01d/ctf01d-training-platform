class AddPlanningWindowsToGames < ActiveRecord::Migration[8.0]
  def change
    add_column :games, :registration_opens_at, :datetime
    add_column :games, :registration_closes_at, :datetime
    add_column :games, :scoreboard_opens_at, :datetime
    add_column :games, :scoreboard_closes_at, :datetime
  end
end

