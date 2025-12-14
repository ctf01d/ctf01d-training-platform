module GameTeamsHelper
  def team_type_badge(game_team)
    return nil unless game_team&.team_type.present?

    cls = case game_team.team_type.to_s.downcase
    when "blue"
      "badge--blue"
    when "red"
      "badge--red"
    else
      "badge--gray"
    end

    content_tag(:span, game_team.team_type, class: "badge #{cls}", style: "margin-left:6px;")
  end
end
