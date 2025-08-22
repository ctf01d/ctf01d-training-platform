class TeamMembership < ApplicationRecord
  belongs_to :team
  belongs_to :user

  ROLE_OWNER        = "owner".freeze
  ROLE_CAPTAIN      = "captain".freeze
  ROLE_VICE_CAPTAIN = "vice_captain".freeze
  ROLE_PLAYER       = "player".freeze
  ROLE_GUEST        = "guest".freeze

  ROLES = [
    ROLE_OWNER,
    ROLE_CAPTAIN,
    ROLE_VICE_CAPTAIN,
    ROLE_PLAYER,
    ROLE_GUEST
  ].freeze

  STATUS_PENDING  = "pending".freeze
  STATUS_APPROVED = "approved".freeze
  STATUS_REJECTED = "rejected".freeze

  STATUSES = [
    STATUS_PENDING,
    STATUS_APPROVED,
    STATUS_REJECTED
  ].freeze

  # Роли, имеющие право управлять командой
  def self.manager_roles
    [ ROLE_OWNER, ROLE_CAPTAIN, ROLE_VICE_CAPTAIN ]
  end

  validates :role, inclusion: { in: ROLES }
  validates :status, inclusion: { in: STATUSES }
end
