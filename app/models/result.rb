class Result < ApplicationRecord
  belongs_to :game
  belongs_to :team

  validates :score, numericality: { only_integer: true }
  validates :game_id, presence: true
  validates :team_id, presence: true
  validates :team_id, uniqueness: { scope: :game_id, message: "уже имеет результат для этой игры" }

  after_create :ensure_game_team_link
  after_update :ensure_game_team_link, if: :saved_change_to_team_id?

  private

  def ensure_game_team_link
    return unless game_id.present? && team_id.present?

    gt = GameTeam.find_or_initialize_by(game_id: game_id, team_id: team_id)
    if gt.new_record?
      next_order = GameTeam.where(game_id: game_id).maximum(:order).to_i + 1
      gt.order = next_order
    end
    gt.save! if gt.changed? || gt.new_record?
  end
end
