class Game < ApplicationRecord
  has_many :results, dependent: :destroy
  has_many :teams, through: :results

  validates :name, presence: true
end
