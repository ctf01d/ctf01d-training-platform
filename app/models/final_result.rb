class FinalResult < ApplicationRecord
  belongs_to :game
  belongs_to :team

  validates :game_id, presence: true
  validates :team_id, presence: true
  validates :score, numericality: { only_integer: true }, allow_nil: true
  validates :position, numericality: { only_integer: true }, allow_nil: true
  validates :team_id, uniqueness: { scope: :game_id }
end

