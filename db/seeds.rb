# Seed: users
require 'set'
require 'erb'
require 'zlib'
require 'time'

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
  'Новосибирский государственный университет (НГУ, NSU)',
  'Московский физико-технический институт',
  'Национальный исследовательский ядерный университет МИФИ',
  'Бауманский МГТУ',
  'Университет ИТМО',
  'Уральский федеральный университет',
  'Казанский федеральный университет',
  'Дальневосточный федеральный университет',
  'Высшая школа экономики',
  'Алтайский государственный технический университет (АлтГТУ, AltSTU)',
  'Томский государственный университет систем управления и радиоэлектроники (ТУСУР, TUSUR)',
  'Томский государственный университет (ТГУ, TSU)',
  'Сибирский государственный университет геосистем и технологий (ССУГиТ, SSUGT)',
  'Новосибирский государственный технический университет (НГТУ, NSTU)'
]

universities = university_names.map do |name|
  u = University.find_or_create_by!(name: name)
  if u.avatar_url.blank?
    u.update!(avatar_url: svg_data_avatar(name, PALETTE.sample))
  end
  u
end

teams = []

# Seed: CyberSibir 2025 teams (ctf01d config)
cybersibir_teams_data = [
  { config_id: 't01', name: 'QarabagTeam',      ip_address: '10.10.1.3',  logo: './html/images/teams/team01.png', active: true },
  { config_id: 't02', name: 'W@zz4b1',          ip_address: '10.10.2.3',  logo: './html/images/teams/team02.png', active: true },
  { config_id: 't03', name: 'smiley-from-telega', ip_address: '10.10.3.3', logo: './html/images/teams/team03.png', active: true },
  { config_id: 't04', name: 'R3T4RD0Z',         ip_address: '10.10.4.3',  logo: './html/images/teams/team04.png', active: true },
  { config_id: 't05', name: 'химозный рулет',   ip_address: '10.10.5.3',  logo: './html/images/teams/team05.png', active: true },
  { config_id: 't06', name: 'SiBears',          ip_address: '10.10.6.3',  logo: './html/images/teams/team06.png', active: true },
  { config_id: 't07', name: 'kekw',             ip_address: '10.10.7.3',  logo: './html/images/teams/team07.png', active: true },
  { config_id: 't08', name: 'CyberCringe',      ip_address: '10.10.8.3',  logo: './html/images/teams/team08.png', active: true },
  { config_id: 't09', name: 'NFB',              ip_address: '10.10.9.3',  logo: './html/images/teams/team09.png', active: true },
  { config_id: 't10', name: 'SharLike',         ip_address: '10.10.10.3', logo: './html/images/teams/team10.png', active: true },
  { config_id: 't11', name: 'ScareCrow',        ip_address: '10.10.11.3', logo: './html/images/teams/team11.png', active: true },
  { config_id: 't12', name: 'd34dl1n3',         ip_address: '10.10.12.3', logo: './html/images/teams/team12.png', active: true },
  { config_id: 't13', name: 'Keva',             ip_address: '10.10.13.3', logo: './html/images/teams/team13.png', active: true },
  { config_id: 't14', name: '4Ray',             ip_address: '10.10.14.3', logo: './html/images/teams/team14.png', active: true },
  { config_id: 't15', name: 'N0N@me13',         ip_address: '10.10.15.3', logo: './html/images/teams/team15.png', active: true },
  { config_id: 't16', name: 'XAKCET',           ip_address: '10.10.16.3', logo: './html/images/teams/team16.png', active: true },
  { config_id: 't17', name: 'TyumGUard',        ip_address: '10.10.17.3', logo: './html/images/teams/team17.png', active: true },
  { config_id: 't18', name: 'datapoison',       ip_address: '10.10.18.3', logo: './html/images/teams/team18.png', active: true },
  { config_id: 't19', name: 'Netoverkill',      ip_address: '10.10.19.3', logo: './html/images/teams/team19.png', active: true },
  { config_id: 't20', name: 'CUT',              ip_address: '10.10.20.3', logo: './html/images/teams/team20.png', active: true },
  { config_id: 't21', name: 'Ibeee',            ip_address: '10.10.21.3', logo: './html/images/teams/team21.png', active: true },
  { config_id: 't22', name: 'o1d_bu7_go1d',     ip_address: '10.10.22.3', logo: './html/images/teams/team22.png', active: true },
  { config_id: 't23', name: 'CyberPatriots',    ip_address: '10.10.23.3', logo: './html/images/teams/team23.png', active: true },
  { config_id: 't24', name: 'vim>nano',         ip_address: '10.10.24.3', logo: './html/images/teams/team24.png', active: true },
  { config_id: 't25', name: 'ГостиИзБудущего',  ip_address: '10.10.25.3', logo: './html/images/teams/team25.png', active: true },
  { config_id: 't26', name: 'Циферки',          ip_address: '10.10.26.3', logo: './html/images/teams/team26.png', active: true },
  { config_id: 't27', name: 'Team #27',         ip_address: '10.10.27.3', logo: './html/images/teams/team27.png', active: true },
  { config_id: 't28', name: 'Team #28',         ip_address: '10.10.28.3', logo: './html/images/teams/team28.png', active: true },
  { config_id: 't29', name: 'Team #29',         ip_address: '10.10.29.3', logo: './html/images/teams/team29.png', active: true }
]

cybersibir_teams = cybersibir_teams_data.map do |attrs|
  t = Team.find_or_initialize_by(name: attrs[:name])
  t.description = [
    'CyberSibir 2025 roster',
    "ctf01d_id: #{attrs[:config_id]}",
    ("IP: #{attrs[:ip_address]}" if attrs[:ip_address]),
    ("Лого: #{attrs[:logo]}" if attrs[:logo]),
    ("Активна: #{attrs[:active] ? 'yes' : 'no'}")
  ].compact.join(' · ')
  t.website = nil
  t.university = nil
  t.avatar_url ||= svg_data_avatar(t.name, PALETTE.sample)
  t.save!
  t
end

teams += cybersibir_teams
cybersibir_team_ids = cybersibir_teams.map(&:id)

# SibirCTF 2018 roster (из jury config)
sibir2018_teams_data = [
  { config_id: 'team1', name: 'Life', logo: 'images/teams/life.jpg', ip_address: '10.218.1.2', active: true },
  { config_id: 'team2', name: 'Void*', logo: 'images/teams/void_.jpg', ip_address: '10.218.2.2', active: true },
  { config_id: 'team3', name: 'SiBears', logo: 'images/teams/sibears.png', ip_address: '10.218.3.2', active: true },
  { config_id: 'team4', name: 'Novosibirsk SU X', logo: 'images/teams/unknown.png', ip_address: '10.218.4.2', active: true },
  { config_id: 'team5', name: 'paperwhale', logo: 'images/teams/paperwhale.png', ip_address: '10.218.5.2', active: true },
  { config_id: 'team6', name: 'Omaviat', logo: 'images/teams/unknown.png', ip_address: '10.218.6.2', active: true },
  { config_id: 'team7', name: 'CatchFM', logo: 'images/teams/unknown.png', ip_address: '10.218.7.2', active: true },
  { config_id: 'team8', name: 'RWX', logo: 'images/teams/rwx.png', ip_address: '10.218.8.2', active: true },
  { config_id: 'team9', name: 'SharLike', logo: 'images/teams/sharlike.png', ip_address: '10.218.9.2', active: true },
  { config_id: 'team10', name: 'd34dl1n3', logo: 'images/teams/d34dl1n3.png', ip_address: '10.218.10.2', active: true },
  { config_id: 'team11', name: 'n57u n00bz', logo: 'images/teams/n57u_n00bz.png', ip_address: '10.218.11.2', active: true },
  { config_id: 'team12', name: 'VoidHack', logo: 'images/teams/voidhack.png', ip_address: '10.218.12.2', active: true },
  { config_id: 'team13', name: "Новосибирский Д'Артаньян", logo: 'images/teams/unknown.png', ip_address: '10.218.13.2', active: true },
  { config_id: 'team14', name: 'Trash Querty', logo: 'images/teams/trash_querty.jpg', ip_address: '10.218.14.2', active: true },
  { config_id: 'team15', name: 'Life (Guest)', logo: 'images/teams/life.jpg', ip_address: '10.218.15.2', active: true },
  { config_id: 'team16', name: 'HawkSquad', logo: 'images/teams/hawk.png', ip_address: '10.218.16.2', active: true },
  # team17 пустое имя и inactive — пропускаем
  { config_id: 'team18', name: 'NeosFun', logo: 'images/teams/neosfun.png', ip_address: '10.218.18.2', active: true }
]

sibir2018_teams = sibir2018_teams_data.filter { |t| t[:name].present? }.map do |attrs|
  t = Team.find_or_initialize_by(name: attrs[:name])
  t.description = [
    'SibirCTF 2018 roster',
    "ctf01d_id: #{attrs[:config_id]}",
    ("IP: #{attrs[:ip_address]}" if attrs[:ip_address]),
    ("Лого: #{attrs[:logo]}" if attrs[:logo]),
    ("Активна: #{attrs[:active] ? 'yes' : 'no'}")
  ].compact.join(' · ')
  t.avatar_url ||= svg_data_avatar(t.name, PALETTE.sample)
  t.save!
  t
end

teams += sibir2018_teams
sibir2018_team_ids = sibir2018_teams.map(&:id)

# SibirCTF 2015 roster (по итогам статьи)
sibir2015_team_names = [
  'SuSlo.PAS',
  'Failers',
  'FTS',
  'Life',
  'Mustang',
  'OMAVIAT',
  'Sharlike',
  'SibirTSU',
  'Zanyato',
  'TIO',
  'Luck3rz',
  'Shikata ga nai',
  'Hell ZIP',
  'n57u n00bz'
]

sibir2015_teams = sibir2015_team_names.map do |name|
  t = Team.find_or_initialize_by(name: name)
  t.description = [ t.description.presence, 'SibirCTF 2015 roster' ].compact.join(' · ')
  t.avatar_url ||= svg_data_avatar(t.name, PALETTE.sample)
  t.save!
  t
end

teams += sibir2015_teams
sibir2015_team_ids = sibir2015_teams.map(&:id)

# SibirCTF 2014 roster (по графику флагов)
sibir2014_teams_data = [
  { name: 'h34dump',              score: 1501 },
  { name: 'Yozik',                score: 1163 },
  { name: 'Brizz',                score: 659 },
  { name: 'Mustang',              score: 626 },
  { name: 'Сборная АлтГТУ',      score: 476 },
  { name: 'Life',                 score: 318 }
]

sibir2014_teams = sibir2014_teams_data.map do |attrs|
  t = Team.find_or_initialize_by(name: attrs[:name])
  t.description = [ t.description.presence, 'SibirCTF 2014 roster' ].compact.join(' · ')
  t.avatar_url ||= svg_data_avatar(t.name, PALETTE.sample)
  t.save!
  t
end

teams += sibir2014_teams
sibir2014_team_ids = sibir2014_teams.map(&:id)

# Маппинг метаданных ctf01d (id/ip) по имени команды
ctf_team_meta = {}
(cybersibir_teams_data + sibir2018_teams_data).each do |attrs|
  ctf_team_meta[attrs[:name].downcase] = { ctf01d_id: attrs[:config_id], ip_address: attrs[:ip_address] }
end

# Привязки команд к университетам (ручные корректировки)
altstu = universities.find { |u| u.name.include?('AltSTU') }
if altstu
  if (sharlike = Team.find_by(name: 'SharLike'))
    desc_parts = sharlike.description.to_s.split(' · ').map(&:strip)
    desc_parts << 'Academic team AltSTU' unless desc_parts.any? { |p| p =~ /AltSTU/i }
    sharlike.update!(university: altstu, description: desc_parts.uniq.join(' · '))
  end
end

tusur = universities.find { |u| u.name =~ /ТУСУР|TUSUR/i }
if tusur
  if (keva = Team.find_by(name: 'Keva'))
    desc_parts = keva.description.to_s.split(' · ').map(&:strip)
    desc_parts << 'Academic team Tomsk State University of Control Systems and Radioelectronics (TUSUR)'
    keva.update!(university: tusur, description: desc_parts.uniq.join(' · '))
  end
end

tsu = universities.find { |u| u.name =~ /ТГУ|TSU/i }
if tsu
  if (sibears = Team.find_by(name: 'SiBears'))
    desc_parts = sibears.description.to_s.split(' · ').map(&:strip)
    desc_parts << 'Academic team TSU'
    desc_parts << 'Томского государственного университета'
    sibears.update!(university: tsu, description: desc_parts.uniq.join(' · '))
  end
end

ssugt = universities.find { |u| u.name =~ /ССУГ|SSUGT|геосистем/i }
if ssugt
  if (dteam = Team.find_by(name: 'd34dl1n3'))
    desc_parts = dteam.description.to_s.split(' · ').map(&:strip)
    desc_parts << 'Academic team SSUGT'
    dteam.update!(university: ssugt, description: desc_parts.uniq.join(' · '), website: dteam.website.presence || 'https://sgugit.ru')
  end
end

nstu = universities.find { |u| u.name =~ /НГТУ|NSTU/i }
if nstu
  if (qarabag = Team.find_by(name: 'QarabagTeam'))
    desc_parts = qarabag.description.to_s.split(' · ').map(&:strip)
    desc_parts << 'Academic team NSTU'
    qarabag.update!(university: nstu, description: desc_parts.uniq.join(' · '), website: qarabag.website.presence || 'https://nstu.ru/')
  end
end

nsu = universities.find { |u| u.name =~ /НГУ|NSU/i }
if nsu
  if (suslo = Team.find_by(name: 'SuSlo.PAS'))
    desc_parts = suslo.description.to_s.split(' · ').map(&:strip)
    aliases = [ 'SUSlo', 'EpicFairPlay', 'ШАПКА ПЕТУХА', 'ОМСКИЙ АНДЕГРАУНД И КУБОК ПЕТУХА', '地松鼠.PAS', '新西伯利亚地松鼠.PAS', 'HoBoCu6uPcKuu NPC', 'Zorro.PAS', 'Большой хомяк выходного дня точка PAS' ]
    desc_parts << "Also known as: #{aliases.join(', ')}"
    desc_parts << 'Academic team Novosibirsk State University (NSU)'
    suslo.update!(university: nsu, description: desc_parts.uniq.join(' · '))
  end
end

# Assign memberships, roles, and captains
used_captain_user_ids = Team.where.not(captain_id: nil).pluck(:captain_id).to_set

teams.each_with_index do |team, idx|
  next if cybersibir_team_ids.include?(team.id) || sibir2018_team_ids.include?(team.id) || sibir2015_team_ids.include?(team.id) || sibir2014_team_ids.include?(team.id) # не добавляем фейковых игроков в предзаполненные ростеры

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
  { name: 'SibirCTF 2014',   organizer: 'keva',    starts_at: Time.utc(2014, 4, 19, 6, 0, 0),  ends_at: Time.utc(2014, 4, 19, 14, 0, 0), site_url: 'https://sibirctf.org/' },
  { name: 'SibirCTF 2015',   organizer: 'keva',    starts_at: Time.utc(2015, 4, 18, 6, 0, 0),  ends_at: Time.utc(2015, 4, 18, 14, 0, 0), site_url: 'https://sibirctf.org/', logo_url: 'https://sun9-29.userapi.com/s/v1/ig1/YxcZz4g9cU0748u9NKGxsxJPwdJ7j6mRYNpsHKZwrYuncf_UVOtVmPPVgkH7SGOgFyluzE5c.jpg?quality=96&as=32x32,48x48,72x72,108x108,160x160,240x240,360x360,480x480,534x534&from=bu&cs=534x0' },
  { name: 'SibirCTF 2016',   organizer: 'keva',    starts_at: Time.utc(2016, 4, 23, 6, 0, 0),  ends_at: Time.utc(2016, 4, 23, 14, 0, 0), site_url: 'https://sibirctf.org/', ctftime_url: 'https://ctftime.org/event/362/' },
  { name: 'SibirCTF 2017',   organizer: 'keva',    starts_at: Time.utc(2017, 4, 22, 6, 0, 0),  ends_at: Time.utc(2017, 4, 22, 14, 0, 0), site_url: 'https://sibirctf.org/' },
  { name: 'SibirCTF 2018',   organizer: 'keva',    starts_at: Time.utc(2018, 10, 21, 4, 0, 0), ends_at: Time.utc(2018, 10, 21, 12, 30, 0), site_url: 'https://sibirctf.org/' },
  { name: 'SibirCTF 2019',   organizer: 'keva',    starts_at: Time.iso8601('2019-11-01T02:00:00Z'), ends_at: Time.iso8601('2019-11-01T12:00:00Z'), site_url: 'https://sibirctf.org/', ctftime_url: 'https://ctftime.org/event/889/', logo_url: 'https://ctftime.org/media/events/sibir_logo.png' },
  { name: 'SibirCTF 2023',   organizer: 'keva',    starts_at: Time.utc(2023, 11, 19, 5, 45, 0), ends_at: Time.utc(2023, 11, 19, 13, 0, 0), site_url: 'https://vk.com/sibirctf', ctftime_url: 'https://ctftime.org/event/2132/', logo_url: 'https://ctftime.org/media/events/glaz2023.jpg' },
  { name: 'CyberSibir 2025', organizer: 'keva',    starts_at: Time.utc(2025, 3, 28, 4, 20, 0), ends_at: Time.utc(2025, 3, 28, 12, 20, 0), site_url: 'https://vk.com/sibirctf', ctftime_url: 'https://ctftime.org/event/2742/', logo_url: 'https://ctftime.org/media/events/cybersibir2025logo_1.png' },
  { name: 'CyberSibir 2026', organizer: 'keva',  starts_at: Time.utc(2026, 3, 27, 4, 0, 0),  ends_at: Time.utc(2026, 3, 27, 12, 0, 0), site_url: 'https://vk.com/sibirctf' }
]

games = games_data.map do |attrs|
  g = Game.find_or_initialize_by(name: attrs[:name])
  g.organizer = attrs[:organizer]
  g.starts_at = attrs[:starts_at]
  g.ends_at = attrs[:ends_at]
  g.site_url = attrs[:site_url] if attrs[:site_url]
  g.ctftime_url = attrs[:ctftime_url] if attrs[:ctftime_url]
  g.avatar_url = attrs[:logo_url] || g.avatar_url || svg_data_avatar(g.name, PALETTE.sample)
  # access/networks demo (публикуем для прошедших и идущих игр; для далёких будущих — оставим пустым)
  if g.ends_at && g.ends_at < Time.now || (g.starts_at && g.starts_at <= Time.now + 2.days)
    slug = g.name.parameterize
    seed_n = Zlib.crc32(slug)
    net_a = 10 + (seed_n % 10)
    net_b = 10 + (seed_n % 200)
    subnet = "10.#{net_a}.#{net_b}.0/24"
    g.vpn_url = "https://vpn.ctf01d.local/#{slug}/connect"
    g.vpn_config_url = "https://vpn.ctf01d.local/#{slug}/#{slug}.ovpn"
    g.access_secret = "DEMO-#{slug.upcase}-#{(seed_n % 1000).to_s.rjust(3, '0')}"
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

# Seed: реальные результаты CyberSibir 2025 (CTFtime)
cyber2025 = games.find { |g| g.name == 'CyberSibir 2025' }
if cyber2025
  scoreboard_2025 = [
    { name: 'TyumGUard',        points: 8746.5 },
    { name: 'smiley-from-telega', points: 8404.6 },
    { name: 'W@zz4b1',          points: 8145.0 },
    { name: 'QarabagTeam',      points: 4646.7 },
    { name: 'химозный рулет',   points: 4595.1 },
    { name: 'datapoison',       points: 4578.5 },
    { name: '4Ray',             points: 4476.9 },
    { name: 'SiBears',          points: 4407.6 },
    { name: 'N0N@me13',         points: 4350.8 },
    { name: 'CUT',              points: 4318.1 },
    { name: 'Ibeee',            points: 3958.7 },
    { name: 'vim>nano',         points: 3224.9 },
    { name: 'NFB',              points: 3190.0 },
    { name: 'o1d_bu7_go1d',     points: 2891.9 },
    { name: 'Циферки',          points: 2693.1 },
    { name: 'ScareCrow',        points: 2567.4 },
    { name: 'd34dl1n3',         points: 2451.2 },
    { name: 'SharLike',         points: 1826.1 },
    { name: 'R3T4RD0Z',         points: 1518.7 },
    { name: 'Netoverkill',      points: 1498.4 },
    { name: 'kekw',             points: 1442.8 },
    { name: 'CyberCringe',      points: 1390.8 },
    { name: 'CyberPatriots',    points: 1223.8 },
    { name: 'XAKCET',           points: 810.4 }
  ]

  team_index = Team.all.index_by { |t| t.name.downcase }
  scoreboard_2025.each do |row|
    team = team_index[row[:name].downcase]
    unless team
      team = Team.create!(
        name: row[:name],
        description: 'Импортировано из scoreboard CyberSibir 2025',
        avatar_url: svg_data_avatar(row[:name], PALETTE.sample)
      )
      team_index[team.name.downcase] = team
      teams << team
    end
    result = Result.find_or_initialize_by(game_id: cyber2025.id, team_id: team.id)
    result.score = (row[:points] * 1000).to_i # сохраняем точность до тысячных
    result.save!
  end
end

# Seed: реальные результаты SibirCTF 2023 (CTFtime)
cyber2023 = games.find { |g| g.name == 'SibirCTF 2023' }
if cyber2023
  scoreboard_2023 = [
    { name: 'SiBears',                  points: 7893.1 },
    { name: 'ыыыыЫЫЫЫЫ',                points: 7386.4 },
    { name: 'CubaLibre',                points: 5832.6 },
    { name: 'QarabagTeam',              points: 5570.8 },
    { name: 'Продам гараж за флаги',    points: 5511.9 },
    { name: 'o1d_bu7_go1d',             points: 4528.1 },
    { name: 'SharLike',                 points: 2817.4 },
    { name: 'd34dl1n3',                 points: 2658.4 },
    { name: 'A4PT Reshetneva',          points: 2569.5 },
    { name: 'ИнфоБесы',                 points: 1783.5 },
    { name: 'LCD',                      points: 897.7 }
  ]

  team_index = Team.all.index_by { |t| t.name.downcase }
  scoreboard_2023.each do |row|
    team = team_index[row[:name].downcase]
    unless team
      team = Team.create!(
        name: row[:name],
        description: 'Импортировано из scoreboard SibirCTF 2023',
        avatar_url: svg_data_avatar(row[:name], PALETTE.sample)
      )
      team_index[team.name.downcase] = team
      teams << team
    end
    result = Result.find_or_initialize_by(game_id: cyber2023.id, team_id: team.id)
    result.score = (row[:points] * 1000).to_i
    result.save!
  end
end

# Seed: реальные результаты SibirCTF 2019 (CTFtime)
cyber2019 = games.find { |g| g.name == 'SibirCTF 2019' }
if cyber2019
  scoreboard_2019 = [
    { name: 'Суслобатя',   points: 89430.9 },
    { name: 'Dragon Hat',  points: 88713.5 },
    { name: 'Tanuki squad', points: 55170.2 },
    { name: 'SiBears',     points: 51399.1 },
    { name: 'Omaviat',     points: 47788.8 },
    { name: 'SharNear',    points: 42291.8 },
    { name: 'rwx',         points: 41812.4 },
    { name: 'UkVQ',        points: 36416.3 },
    { name: '4ерниkа',     points: 32676.4 },
    { name: 'Keva19',      points: 32497.7 },
    { name: 'a-cool-team', points: 26495.9 },
    { name: 'Life',        points: 26139.0 },
    { name: 'd34dl1n3',    points: 24739.2 },
    { name: 'CatchFM',     points: 24347.0 },
    { name: 'BANOЧKA',     points: 6261.2 },
    { name: 'Team 16',     points: 4727.0 }
  ]

  team_index = Team.all.index_by { |t| t.name.downcase }
  scoreboard_2019.each do |row|
    team = team_index[row[:name].downcase]
    unless team
      team = Team.create!(
        name: row[:name],
        description: 'Импортировано из scoreboard SibirCTF 2019',
        avatar_url: svg_data_avatar(row[:name], PALETTE.sample)
      )
      team_index[team.name.downcase] = team
      teams << team
    end
    result = Result.find_or_initialize_by(game_id: cyber2019.id, team_id: team.id)
    result.score = (row[:points]).to_i # очки целые в CTFTIME
    result.save!
  end
end

# Seed: реальные результаты SibirCTF 2016 (CTFtime)
cyber2016 = games.find { |g| g.name == 'SibirCTF 2016' }
if cyber2016
  scoreboard_2016 = [
    { name: 'SiBears',                         points: 3250.770 },
    { name: 'Yozik',                           points: 18.790 },
    { name: 'Team Information Offensive',      points: 45.940 },
    { name: 'FoXXXeS',                         points: 304.700 },
    { name: 'Mu574n9',                         points: 359.700 },
    { name: '!2day',                           points: 982.820 },
    { name: 'SharLike',                        points: 1186.730 },
    { name: '(_xXx_-=HOBOCu6uPCKuE_IICbl_1337=-_xXx_)', points: 1436.590 },
    { name: 'Life',                            points: 1753.340 },
    { name: 'xXx_Я_не_ХЛЕБ_я_КОТ_хХх',         points: 1763.670 },
    { name: 'paperwhale',                      points: 0.810 }
  ]

  team_index = Team.all.index_by { |t| t.name.downcase }
  scoreboard_2016.each do |row|
    team = team_index[row[:name].downcase]
    unless team
      team = Team.create!(
        name: row[:name],
        description: 'Импортировано из scoreboard SibirCTF 2016',
        avatar_url: svg_data_avatar(row[:name], PALETTE.sample)
      )
      team_index[team.name.downcase] = team
      teams << team
    end
    result = Result.find_or_initialize_by(game_id: cyber2016.id, team_id: team.id)
    result.score = (row[:points] * 1000).to_i
    result.save!
  end
end

# Seed: реальные результаты SibirCTF 2015 (из публикации)
cyber2015 = games.find { |g| g.name == 'SibirCTF 2015' }
if cyber2015
  # порядок из статьи; баллы задаём убывающими для сохранения ранга
  scoreboard_2015 = [
    'SuSlo.PAS',
    'Failers',
    'FTS',
    'Life',
    'Mustang',
    'OMAVIAT',
    'Sharlike',
    'SibirTSU',
    'Zanyato',
    'TIO',
    'Luck3rz',
    'Shikata ga nai',
    'Hell ZIP',
    'n57u n00bz'
  ]

  team_index = Team.all.index_by { |t| t.name.downcase }
  base_score = 14000
  step = 600

  scoreboard_2015.each_with_index do |name, idx|
    team = team_index[name.downcase]
    unless team
      team = Team.create!(
        name: name,
        description: 'Импортировано из scoreboard SibirCTF 2015',
        avatar_url: svg_data_avatar(name, PALETTE.sample)
      )
      team_index[team.name.downcase] = team
      teams << team
    end

    score = [ base_score - step * idx, step ].max
    result = Result.find_or_initialize_by(game_id: cyber2015.id, team_id: team.id)
    result.score = score
    result.save!
  end
end

# Seed: результаты SibirCTF 2018 (по финальному scoreboard)
cyber2018 = games.find { |g| g.name == 'SibirCTF 2018' }
if cyber2018
  scoreboard_2018 = [
    { name: 'Новосиб',      score: 7760.15 },
    { name: 'SharLike',     score: 4450.17 },
    { name: 'VoidHack',     score: 4028.91 },
    { name: 'SiBears',      score: 3602.50 },
    { name: 'Novosibir',    score: 1736.33 },
    { name: 'HawkSqu',      score: 1086.32 },
    { name: 'Void*',        score: 1130.22 },
    { name: 'RWX',          score: 1068.26 },
    { name: 'NeosFun',      score: 1047.88 },
    { name: 'Life (Guest)', score: 932.49 },
    { name: 'CatchFM',      score: 903.86 },
    { name: 'paperwhale',   score: 890.19 },
    { name: 'd34dl1n3',     score: 829.86 },
    { name: 'Omaviat',      score: 780.55 },
    { name: 'Life',         score: 778.84 },
    { name: 'Trash Querty', score: 618.19 },
    { name: 'n57u n00bz',   score: 390.13 }
  ]

  alias_to_team = {
    'новосиб' => "Новосибирский Д'Артаньян",
    'novosibir' => 'Novosibirsk SU X',
    'hawksqu' => 'HawkSquad',
    'paperwhale' => 'paperwhale',
    'trash querty' => 'Trash Querty',
    'life (guest)' => 'Life (Guest)'
  }

  team_index = Team.all.index_by { |t| t.name.downcase }
  scoreboard_2018.each do |row|
    lookup = alias_to_team[row[:name].downcase] || row[:name]
    team = team_index[lookup.downcase]
    unless team
      team = Team.create!(
        name: lookup,
        description: 'Импортировано из scoreboard SibirCTF 2018',
        avatar_url: svg_data_avatar(lookup, PALETTE.sample)
      )
      team_index[team.name.downcase] = team
      teams << team
    end
    result = Result.find_or_initialize_by(game_id: cyber2018.id, team_id: team.id)
    result.score = row[:score].to_i
    result.save!
  end
end

# Seed: результаты SibirCTF 2014 (по графику флагов)
cyber2014 = games.find { |g| g.name == 'SibirCTF 2014' }
if cyber2014
  scoreboard_2014 = [
    { name: 'h34dump', score: 1501 },
    { name: 'Yozik', score: 1163 },
    { name: 'Brizz', score: 659 },
    { name: 'Mustang', score: 626 },
    { name: 'Сборная АлтГТУ', score: 476 },
    { name: 'Life', score: 318 }
  ]

  team_index = Team.all.index_by { |t| t.name.downcase }
  scoreboard_2014.each_with_index do |row, idx|
    team = team_index[row[:name].downcase]
    unless team
      team = Team.create!(
        name: row[:name],
        description: 'Импортировано из scoreboard SibirCTF 2014',
        avatar_url: svg_data_avatar(row[:name], PALETTE.sample)
      )
      team_index[team.name.downcase] = team
      teams << team
    end
    result = Result.find_or_initialize_by(game_id: cyber2014.id, team_id: team.id)
    result.score = row[:score]
    result.save!
  end
end

# GameTeams: создаём связи игра↔команда с порядком и ctf01d метаданными
Game.find_each do |g|
  results = g.results.includes(:team).order(score: :desc, id: :asc).to_a
  results.each_with_index do |r, idx|
    next unless r.team
    meta = ctf_team_meta[r.team.name.downcase] || {}
    gt = GameTeam.find_or_initialize_by(game_id: g.id, team_id: r.team_id)
    gt.order = idx + 1 if gt.order.to_i.zero?
    gt.ctf01d_id ||= meta[:ctf01d_id]
    gt.ip_address ||= meta[:ip_address]
    gt.save!
  end
end

# Seed: services (SibirCTF/CyberSibir)
services_data = [
  # CyberSibir 2025
  { name: 'EyeSee',       public_description: 'Service with vulnerabilities for CyberSibir 2025', author: 'CyberSibir', repo: '2025-cybersibir-service-eyesee', language: 'Python', games: [ 'CyberSibir 2025' ] },
  { name: 'MSPD',         public_description: 'Service with vulnerabilities for CyberSibir 2025', author: 'CyberSibir', repo: '2025-cybersibir-service-mspd', language: 'Go', games: [ 'CyberSibir 2025' ] },
  { name: 'NcDEx',        public_description: 'Service with vulnerabilities for CyberSibir 2025', author: 'CyberSibir', repo: '2025-cybersibir-service-ncdex', language: 'Crystal', games: [ 'CyberSibir 2025' ] },
  { name: 'Unpleasant',   public_description: 'Service with vulnerabilities for CyberSibir 2025', author: 'CyberSibir', repo: '2025-cybersibir-service-unpleasant', language: 'HTML', games: [ 'CyberSibir 2025' ] },
  { name: 'WrNum',        public_description: 'Service with vulnerabilities for CyberSibir 2025', author: 'CyberSibir', repo: '2025-cybersibir-service-wrnum', language: 'Python', games: [ 'CyberSibir 2025' ] },
  { name: 'CyberBank',    public_description: 'Service with vulnerabilities for CyberSibir 2025', author: 'CyberSibir', repo: '2025-cybersibir-service-bank', language: 'JavaScript', games: [ 'CyberSibir 2025' ] },
  { name: 'NeuroLink234', public_description: 'Service with vulnerabilities for CyberSibir 2025', author: 'CyberSibir', repo: '2025-cybersibir-service-neLi234', language: 'C', games: [ 'CyberSibir 2025' ] },
  { name: 'BioGuard',     public_description: 'Service with vulnerabilities for CyberSibir 2025', author: 'CyberSibir', repo: '2025-cybersibir-service-BioGuard', language: 'Python', games: [ 'CyberSibir 2025' ] },
  # SibirCTF 2023
  { name: 'StickMarket',    public_description: 'SibirCTF 2023 service (StickMarket)', author: 'SibirCTF', repo: '2023-service-sibirctf-stickmarket', language: 'CSS', games: [ 'SibirCTF 2023' ] },
  { name: 'SouthParkChat',  public_description: 'SibirCTF 2023 service (SouthParkChat)', author: 'SibirCTF', repo: '2023-service-sibirctf-southparkchat', language: 'Go', games: [ 'SibirCTF 2023' ] },
  { name: 'SX',             public_description: 'SibirCTF 2023 service (SX)', author: 'SibirCTF', repo: '2023-service-sibirctf-sx', language: 'Python', games: [ 'SibirCTF 2023' ] },
  { name: 'Chef',           public_description: 'SibirCTF 2023 service (Chef)', author: 'SibirCTF', repo: '2023-service-sibirctf-chef', language: 'C', games: [ 'SibirCTF 2023' ] },
  { name: 'Card Vault',     public_description: 'SibirCTF 2023 CardVault service', author: 'SibirCTF', repo: '2023-service-sibirctf-cardvault', language: 'Elixir', games: [ 'SibirCTF 2023' ] },
  # SibirCTF 2018
  { name: 'maxigram',       public_description: 'SibirCTF 2018 service (maxigram)', author: 'SibirCTF', repo: '2018-service-maxigram', language: 'Python', games: [ 'SibirCTF 2018' ] },
  { name: 'The Fakebook',   public_description: 'SibirCTF 2018 service (The Fakebook)', author: 'SibirCTF', repo: '2018-service-thefakebook', language: 'HTML', games: [ 'SibirCTF 2018' ] },
  { name: 'The Hole',       public_description: 'SibirCTF 2018 service (The Hole)', author: 'SibirCTF', repo: '2018-service-the-hole', language: 'C++', games: [ 'SibirCTF 2018' ] },
  { name: 'Legacy News',    public_description: 'SibirCTF 2018 service (Legacy News)', author: 'SibirCTF', repo: nil, language: nil, games: [ 'SibirCTF 2018' ] },
  { name: 'Mirai',          public_description: 'SibirCTF 2018 service (Mirai)', author: 'SibirCTF', repo: '2018-service-mirai', language: 'PHP', games: [ 'SibirCTF 2018' ] },
  { name: 'LNKS',           public_description: 'SibirCTF 2018 service (LNKS)', author: 'SibirCTF', repo: '2018-service-lnks', language: 'PHP', games: [ 'SibirCTF 2018' ] },
  { name: 'Lie2Me',         public_description: 'SibirCTF 2018 service (Lie2Me)', author: 'SibirCTF', repo: '2018-service-lie-to-me', language: 'Perl', games: [ 'SibirCTF 2018' ] },
  # SibirCTF 2015
  { name: 'CryChat',        public_description: 'SibirCTF 2015 service (CryChat)', author: 'SibirCTF', repo: '2015-crychat', language: 'PHP', games: [ 'SibirCTF 2015' ] },
  { name: "O'Foody",        public_description: 'SibirCTF 2015 service (O’Foody)', author: 'SibirCTF', repo: '2015-ofoody', language: 'Perl', games: [ 'SibirCTF 2015' ] },
  { name: 'CTFGram',        public_description: 'SibirCTF 2015 service (CTFGram)', author: 'SibirCTF', repo: '2015-ctfgram', language: 'JavaScript', games: [ 'SibirCTF 2015' ] },
  { name: 'EasyAs',         public_description: 'SibirCTF 2015 service (EasyAs)', author: 'SibirCTF', repo: '2015-easyas', language: 'Python', games: [ 'SibirCTF 2015' ] }
]

services = services_data.map do |attrs|
  Service.find_or_create_by!(name: attrs[:name]) do |s|
    s.public_description = attrs[:public_description]
    s.private_description = ""
    s.author = attrs[:author]
    s.avatar_url = svg_data_avatar(attrs[:name], PALETTE.sample)
    s.public = true
    s.service_archive_url = attrs[:repo] ? "https://github.com/SibirCTF/#{attrs[:repo]}/archive/refs/heads/main.zip" : nil
    s.checker_archive_url = nil
    s.writeup_url = nil
    s.exploits_url = nil
    s.check_status = 'unknown'
    s.checked_at = nil
  end.tap do |s|
    s.update!(language: attrs[:language]) if s.respond_to?(:language=) && attrs[:language]
  end
end

# Assign services to games (детерминированно)
services_by_game = Hash.new { |h, k| h[k] = [] }
services_data.each do |svc|
  svc[:games].each { |g| services_by_game[g] << svc[:name] }
end

games.each do |g|
  names = services_by_game[g.name] || []
  g.services = Service.where(name: names)
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
