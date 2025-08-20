class TeamMembership < ApplicationRecord
  belongs_to :team
  belongs_to :user

  ROLES = %w[owner captain vice_captain player guest].freeze
  STATUSES = %w[pending approved rejected].freeze

  validates :role, inclusion: { in: ROLES }
  validates :status, inclusion: { in: STATUSES }
end
