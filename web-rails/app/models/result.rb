class Result < ApplicationRecord
  belongs_to :game
  belongs_to :team

  validates :score, numericality: { only_integer: true }
  validates :game_id, presence: true
  validates :team_id, presence: true
  validates :team_id, uniqueness: { scope: :game_id, message: 'уже имеет результат для этой игры' }
end
