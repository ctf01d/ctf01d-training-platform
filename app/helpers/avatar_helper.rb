module AvatarHelper
  PALETTE = %w[
    #3B82F6 #10B981 #F59E0B #EF4444 #8B5CF6 #06B6D4 #EC4899 #84CC16 #F97316 #22C55E
  ].freeze

  def svg_data_avatar(text, size: 96, bg: nil)
    require 'erb'
    require 'zlib'
    label = text.to_s.strip
    initial = label[0]&.upcase || '?'
    color = bg || PALETTE[Zlib.crc32(label) % PALETTE.length]
    svg = <<~SVG
      <svg xmlns='http://www.w3.org/2000/svg' width='#{size}' height='#{size}'>
        <rect width='100%' height='100%' fill='#{color}' />
        <text x='50%' y='56%' dominant-baseline='middle' text-anchor='middle'
              font-family='Arial, Helvetica, sans-serif' font-size='#{(size*0.5).to_i}' fill='#fff'>#{initial}</text>
      </svg>
    SVG
    "data:image/svg+xml;utf8,#{ERB::Util.url_encode(svg)}"
  end

  def avatar_image_tag(name, url: nil, size: 48, class_name: 'avatar-image')
    src = url.to_s.strip
    if src.present? && (src.start_with?('http://', 'https://', '/') || src.start_with?('data:image'))
      image_tag src, alt: name, class: class_name
    else
      image_tag svg_data_avatar(name, size: size), alt: name, class: class_name
    end
  end
end

