class Service < ApplicationRecord
  has_and_belongs_to_many :games
  validates :name, presence: true, uniqueness: true
  scope :publicly_visible, -> { where(public: true) }
end
