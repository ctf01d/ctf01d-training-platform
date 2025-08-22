class Writeup < ApplicationRecord
  belongs_to :game
  belongs_to :team

  validates :title, presence: true, length: { maximum: 255 }
  validates :url, presence: true, format: { with: %r{\Ahttps?://}i, message: "должен начинаться с http(s)://" }
  validates :team_id, uniqueness: { scope: [ :game_id, :title ], message: "уже есть writeup с таким названием для этой игры" }
end
