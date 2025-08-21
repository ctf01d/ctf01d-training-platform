# Seed: users
admin = User.find_or_initialize_by(user_name: 'admin')
admin.display_name = 'Admin'
admin.role = 'admin'
admin.rating ||= 0
admin.password = 'admin'
admin.password_confirmation = 'admin'
admin.save!

users_data = %w[
  r3v
  heap_wizard
  crypto_kid
  nullbyte
  segfault
  pwnicorn
  xssninja
  forensic_fox
  overflower
  rootkitten
  packet_pirate
  shellsamurai
]

users = users_data.map do |login|
  u = User.find_or_initialize_by(user_name: login)
  u.display_name = login.tr('_', ' ').split.map(&:capitalize).join(' ')
  u.role = 'player'
  u.rating ||= rand(0..200)
  u.password = 'password'
  u.password_confirmation = 'password'
  u.save!
  u
end

# Seed: universities (subset from Go migrations)
university_names = [
  'Московский государственный университет имени М.В. Ломоносова',
  'Санкт-Петербургский государственный университет',
  'Новосибирский государственный университет',
  'Московский физико-технический институт',
  'Национальный исследовательский ядерный университет МИФИ',
  'Бауманский МГТУ',
  'Университет ИТМО',
  'Уральский федеральный университет',
  'Казанский федеральный университет',
  'Дальневосточный федеральный университет',
  'Томский государственный университет',
  'Высшая школа экономики'
]

universities = university_names.map do |name|
  University.find_or_create_by!(name: name)
end

# Seed: teams
teams_data = [
  { name: 'Pwnicorns',        website: 'https://pwnicorns.ctf',        description: 'We pwn for fun and profit.' },
  { name: 'NullByte',         website: 'https://nullbyte.ctf',         description: 'Nothing but 0x00(s).' },
  { name: 'CryptoCats',       website: 'https://cryptocats.ctf',       description: 'Meow-dern cryptography enthusiasts.' },
  { name: 'SegFault Squad',   website: 'https://segfaultsquad.ctf',    description: 'Core dumped since 2013.' },
  { name: 'RedOps',           website: 'https://redops.ctf',           description: 'Offense-first CTF crew.' },
  { name: 'Overflower',       website: 'https://overflower.ctf',       description: 'Grow your stack.' }
]

teams = teams_data.map do |attrs|
  t = Team.find_or_initialize_by(name: attrs[:name])
  t.description = attrs[:description]
  t.website = attrs[:website]
  t.university = [nil, *universities.sample(1)].compact.first
  t.save!
  t
end

# Assign memberships, roles, and captains
used_captain_user_ids = Team.where.not(captain_id: nil).pluck(:captain_id).to_set

teams.each_with_index do |team, idx|
  # Pick 4-6 distinct users for the team
  members = users.sample(4 + (idx % 3))

  # Ensure owner
  owner = members.first
  m_owner = TeamMembership.find_or_initialize_by(team_id: team.id, user_id: owner.id)
  m_owner.role = TeamMembership::ROLE_OWNER
  m_owner.status = TeamMembership::STATUS_APPROVED
  m_owner.save!

  # Other members as players or vice_captain
  members[1..].each_with_index do |u, i|
    m = TeamMembership.find_or_initialize_by(team_id: team.id, user_id: u.id)
    m.role = (i.zero? ? TeamMembership::ROLE_VICE_CAPTAIN : TeamMembership::ROLE_PLAYER)
    m.status = TeamMembership::STATUS_APPROVED
    m.save!
  end

  # Pick captain: prefer someone not already a captain elsewhere
  candidate = (members.find { |u| !used_captain_user_ids.include?(u.id) } || owner)
  unless Team.exists?(captain_id: candidate.id)
    team.update!(captain_id: candidate.id)
    used_captain_user_ids.add(candidate.id)
    # Ensure captain has membership and role updated
    cap_m = TeamMembership.find_or_initialize_by(team_id: team.id, user_id: candidate.id)
    cap_m.role = TeamMembership::ROLE_CAPTAIN
    cap_m.status = TeamMembership::STATUS_APPROVED
    cap_m.save!
  end
end

# Seed: games (past, ongoing, upcoming)
games_data = [
  { name: '0CTF Intergalactic', organizer: 'CTF01D', starts_at: 30.days.ago, ends_at: 29.days.ago },
  { name: 'Hack The Planet',    organizer: 'CTF01D', starts_at: 8.days.ago,  ends_at: 7.days.ago  },
  { name: 'PWN Arena',          organizer: 'CTF01D', starts_at: 1.day.ago,   ends_at: 1.day.from_now },
  { name: 'Crypto Clash',       organizer: 'CTF01D', starts_at: 5.days.from_now, ends_at: 6.days.from_now },
  { name: 'Forensics Frenzy',   organizer: 'CTF01D', starts_at: 15.days.from_now, ends_at: 16.days.from_now }
]

games = games_data.map do |attrs|
  g = Game.find_or_initialize_by(name: attrs[:name])
  g.organizer = attrs[:organizer]
  g.starts_at = attrs[:starts_at]
  g.ends_at = attrs[:ends_at]
  # планирование
  if g.starts_at && g.ends_at
    g.registration_opens_at = g.starts_at - 7.days
    g.registration_closes_at = g.starts_at - 1.day
    g.scoreboard_opens_at = g.starts_at
    g.scoreboard_closes_at = g.ends_at + 7.days
  end
  g.save!
  g
end

# Seed: results for past and ongoing games
srand(42)
games.each do |g|
  next if g.starts_at > Time.now # skip upcoming
  # Rank a random subset of teams (4..teams.size)
  participating = teams.sample([teams.size, 4 + rand(0..(teams.size - 4))].min)
  base = 1000
  step = 75
  participating.shuffle.each_with_index do |t, rank|
    score = [base - step * rank + rand(-20..20), 0].max
    r = Result.find_or_initialize_by(game_id: g.id, team_id: t.id)
    r.score = score
    r.save!
  end
end

# Seed: services (CTF-flavored)
services_data = [
  { name: 'pwn: Baby Heap',            public_description: 'Базовые аллокации/фри, tcache.', author: 'core', public: true },
  { name: 'pwn: ROP Gadgets',          public_description: 'Ret2libc и ROP в x64.', author: 'core', public: true },
  { name: 'pwn: Format String 101',    public_description: 'Базовая FSB, чтение/запись.', author: 'core', public: true },
  { name: 'pwn: Kernel Intro',         public_description: 'Средний уровень: kalloc, uaf.', author: 'core', public: false },

  { name: 'web: Cookie Monster',       public_description: 'XSS, cookies, CSP.', author: 'webops', public: true },
  { name: 'web: SSTI Bakery',          public_description: 'SSTI в шаблонизаторе.', author: 'webops', public: true },
  { name: 'web: SQL Trickery',         public_description: 'Blind SQLi и WAF bypass.', author: 'webops', public: true },
  { name: 'web: Deserialization Mayhem', public_description: 'Insecure deserialization.', author: 'webops', public: false },

  { name: 'crypto: ECB Roulette',      public_description: 'ECB oracle, padding.', author: 'crypto', public: true },
  { name: 'crypto: LCG Madness',       public_description: 'PRNG predictability.', author: 'crypto', public: true },
  { name: 'crypto: RSA Clinic',        public_description: 'Common RSA pitfalls.', author: 'crypto', public: true },

  { name: 're: CrackMe Deluxe',        public_description: 'Reverse basic auth.', author: 'rev', public: true },
  { name: 're: VM Confuser',           public_description: 'VM obfuscation tricks.', author: 'rev', public: false },

  { name: 'forensics: PCAP Hunt',      public_description: 'Разбор сетевого дампа.', author: 'forensics', public: true },
  { name: 'forensics: Disk Image',     public_description: 'Образы и артефакты.', author: 'forensics', public: true },
  { name: 'misc: OSINT Trail',         public_description: 'Поиск по следам.', author: 'misc', public: true },
  { name: 'hardware: UART Quest',      public_description: 'Железо и логи.', author: 'hw', public: false }
]

services = services_data.map do |attrs|
  Service.find_or_create_by!(name: attrs[:name]) do |s|
    s.public_description = attrs[:public_description]
    s.private_description = ""
    s.author = attrs[:author]
    s.avatar_url = ''
    s.public = attrs[:public]
    s.check_status = 'unknown'
  end
end

# Assign services to games
games.each_with_index do |g, i|
  # Upcoming games: fewer services; ongoing/past: more
  count = if g.ends_at && g.ends_at < Time.now
            4
          elsif g.starts_at && g.starts_at <= Time.now && g.ends_at && g.ends_at >= Time.now
            5
          else
            3
          end
  g.services = services.sample(count)
  g.save!
end

# Snapshot final results for past games
Game.where('ends_at < ?', Time.now).find_each do |g|
  next if g.final_results.exists?
  rows = g.results.order(score: :desc).to_a
  rows.each_with_index do |r, idx|
    FinalResult.create!(game_id: g.id, team_id: r.team_id, score: r.score.to_i, position: idx + 1)
  end
  g.update!(finalized: true, finalized_at: Time.now)
end

# Seed: writeups for past games (random teams)
past_games = Game.where('ends_at < ?', Time.now).to_a
titles = [
  'How we pwned Baby Heap',
  'Bypassing CSP in Cookie Monster',
  'ECB Oracle Walkthrough',
  'Reversing CrackMe Deluxe',
  'Forensics PCAP Tips',
  'ROP the Planet'
]
past_games.each do |g|
  teams.sample(2).each_with_index do |t, i|
    Writeup.find_or_create_by!(game: g, team: t, title: titles.sample) do |w|
      w.url = "https://writeups.example/#{t.name.parameterize}-#{g.name.parameterize}-#{i+1}"
    end
  end
end
# Seed: users
require 'set'
