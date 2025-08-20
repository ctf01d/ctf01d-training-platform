class Service < ApplicationRecord
  validates :name, presence: true, uniqueness: true
  scope :publicly_visible, -> { where(public: true) }
end
