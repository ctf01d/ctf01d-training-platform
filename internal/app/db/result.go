package db

type Result struct {
	Id     int    `db:"id"`
	TeamId string `db:"team_id"`
	GameId string `db:"game_id"`
	Score  int    `db:"score"`
	Rank   int    `db:"rank"`
}
