class Result < ApplicationRecord
  belongs_to :game
  belongs_to :team

  validates :score, numericality: { only_integer: true }
end
