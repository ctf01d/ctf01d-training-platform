admin = User.find_or_initialize_by(user_name: 'admin')
admin.display_name = 'Admin'
admin.role = 'admin'
admin.rating ||= 0
admin.password = 'admin'
admin.password_confirmation = 'admin'
admin.save!

uni = University.find_or_create_by!(name: 'Demo University') do |u|
  u.site_url = 'https://example.edu'
end

team = Team.find_or_create_by!(name: 'Team One') do |t|
  t.university = uni
  t.description = 'Demo team'
  t.website = 'https://example.com'
  t.avatar_url = ''
end

Service.find_or_create_by!(name: 'Example Service') do |s|
  s.public_description = 'Публичное описание'
  s.private_description = 'Приватное описание'
  s.author = 'Demo Author'
  s.copyright = 'MIT'
  s.avatar_url = ''
  s.public = true
end

game = Game.find_or_create_by!(name: 'Training Game') do |g|
  g.organizer = 'CTF01D'
  g.starts_at = Time.now
  g.ends_at = 1.day.from_now
end

Result.find_or_create_by!(game: game, team: team) do |r|
  r.score = 0
end
