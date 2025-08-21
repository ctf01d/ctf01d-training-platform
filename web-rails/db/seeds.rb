# Seed: users
require 'set'
require 'erb'
require 'zlib'

# Simple SVG avatar generator (data URL)
def svg_data_avatar(text, bg = '#3B82F6')
  initial = text.to_s[0]&.upcase || '?'
  svg = <<~SVG
    <svg xmlns='http://www.w3.org/2000/svg' width='96' height='96'>
      <rect width='100%' height='100%' fill='#{bg}' />
      <text x='50%' y='56%' dominant-baseline='middle' text-anchor='middle'
            font-family='Arial, Helvetica, sans-serif' font-size='48' fill='#fff'>#{initial}</text>
    </svg>
  SVG
  "data:image/svg+xml;utf8,#{ERB::Util.url_encode(svg)}"
end

PALETTE = %w[
  #3B82F6 #10B981 #F59E0B #EF4444 #8B5CF6 #06B6D4 #EC4899 #84CC16 #F97316 #22C55E
]
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
  bufferbuddha
  crypto_owl
  sqli_master
  ssti_sniffer
  des3
  vmwizard
  sigsegv
  oob_read
  sandboxer
  botnet_bob
  z3solver
  regexploit
]

users = users_data.map do |login|
  u = User.find_or_initialize_by(user_name: login)
  u.display_name = login.tr('_', ' ').split.map(&:capitalize).join(' ')
  u.role = 'player'
  u.rating ||= rand(0..200)
  u.password = 'password'
  u.password_confirmation = 'password'
  u.avatar_url = svg_data_avatar(u.display_name, PALETTE.sample)
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
  u = University.find_or_create_by!(name: name)
  if u.avatar_url.blank?
    u.update!(avatar_url: svg_data_avatar(name, PALETTE.sample))
  end
  u
end

# Seed: teams
teams_data = [
  { name: 'Pwnicorns',        website: 'https://pwnicorns.ctf',        description: 'We pwn for fun and profit.' },
  { name: 'NullByte',         website: 'https://nullbyte.ctf',         description: 'Nothing but 0x00(s).' },
  { name: 'CryptoCats',       website: 'https://cryptocats.ctf',       description: 'Meow-dern cryptography enthusiasts.' },
  { name: 'SegFault Squad',   website: 'https://segfaultsquad.ctf',    description: 'Core dumped since 2013.' },
  { name: 'RedOps',           website: 'https://redops.ctf',           description: 'Offense-first CTF crew.' },
  { name: 'Overflower',       website: 'https://overflower.ctf',       description: 'Grow your stack.' },
  { name: 'BitBenders',       website: 'https://bitbenders.ctf',       description: 'We bend bits to our will.' },
  { name: 'Stack Smashers',   website: 'https://stacksmashers.ctf',    description: 'Mind the canaries.' },
  { name: 'Cyber Owls',       website: 'https://cyberowls.ctf',        description: 'Night hunters of bugs.' },
  { name: 'Ghost in Shellcode', website: 'https://ghostshellcode.ctf', description: 'Haunting binaries since forever.' }
]

teams = teams_data.map do |attrs|
  t = Team.find_or_initialize_by(name: attrs[:name])
  t.description = attrs[:description]
  t.website = attrs[:website]
  t.university = [nil, *universities.sample(1)].compact.first
  t.avatar_url = svg_data_avatar(t.name, PALETTE.sample)
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
  { name: 'Forensics Frenzy',   organizer: 'CTF01D', starts_at: 15.days.from_now, ends_at: 16.days.from_now },
  { name: 'Reversing Rumble',   organizer: 'CTF01D', starts_at: 60.days.ago, ends_at: 58.days.ago },
  { name: 'Binary Blitz',       organizer: 'CTF01D', starts_at: 10.days.from_now, ends_at: 11.days.from_now },
  { name: 'Stego Showdown',     organizer: 'CTF01D', starts_at: 20.days.from_now, ends_at: 20.days.from_now + 12.hours }
]

games = games_data.map do |attrs|
  g = Game.find_or_initialize_by(name: attrs[:name])
  g.organizer = attrs[:organizer]
  g.starts_at = attrs[:starts_at]
  g.ends_at = attrs[:ends_at]
  g.avatar_url = svg_data_avatar(g.name, PALETTE.sample)
  # access/networks demo (публикуем для прошедших и идущих игр; для далёких будущих — оставим пустым)
  if g.ends_at && g.ends_at < Time.now || (g.starts_at && g.starts_at <= Time.now + 2.days)
    slug = g.name.parameterize
    seed_n = Zlib.crc32(slug)
    net_a = 10 + (seed_n % 10)
    net_b = 10 + (seed_n % 200)
    subnet = "10.#{net_a}.#{net_b}.0/24"
    g.vpn_url = "https://vpn.ctf01d.local/#{slug}/connect"
    g.vpn_config_url = "https://vpn.ctf01d.local/#{slug}/#{slug}.ovpn"
    g.access_secret = "DEMO-#{slug.upcase}-#{(seed_n % 1000).to_s.rjust(3,'0')}"
    g.access_instructions = <<~TXT
      Подключитесь к VPN перед атакой на сервисы.

      Вариант 1 — OpenVPN:
      1) Скачайте конфиг: #{g.vpn_config_url}
      2) Импортируйте в OpenVPN и подключитесь.

      Вариант 2 — WireGuard:
      1) Скачайте профиль: https://vpn.ctf01d.local/#{slug}/#{slug}.conf
      2) Импортируйте в WireGuard и подключитесь.

      Логин: team-<team_id>
      Пароль/ключ: узнавайте у капитана или организаторов (секрет: #{g.access_secret}).

      Внутренняя сеть игры: #{subnet}
    TXT
  end
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
    slug = attrs[:name].downcase.gsub(/[^a-z0-9]+/, '-').gsub(/^-|-$/,'')
    s.public_description = attrs[:public_description]
    s.private_description = ""
    s.author = attrs[:author]
    s.avatar_url = svg_data_avatar(attrs[:name], PALETTE.sample)
    s.public = attrs[:public]
    s.service_archive_url = "https://example.com/archives/#{slug}.zip"
    s.checker_archive_url = "https://example.com/checkers/#{slug}-checker.zip"
    s.writeup_url = "https://example.com/writeups/#{slug}.pdf"
    s.exploits_url = (attrs[:name].start_with?('pwn') || attrs[:name].start_with?('web')) ? "https://example.com/exploits/#{slug}.zip" : nil
    status_pool = %w[unknown queued ok fail]
    s.check_status = status_pool.sample
    s.checked_at = Time.now - rand(0..10).days
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
# end of seeds
