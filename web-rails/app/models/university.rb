class University < ApplicationRecord
  has_many :teams

  validates :name, presence: true, uniqueness: true
end
