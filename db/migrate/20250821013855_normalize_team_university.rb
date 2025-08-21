class NormalizeTeamUniversity < ActiveRecord::Migration[8.0]
  class MTeam < ApplicationRecord
    self.table_name = 'teams'
  end

  class MUniversity < ApplicationRecord
    self.table_name = 'universities'
  end

  def up
    add_reference :teams, :university, foreign_key: true, index: true, null: true

    # Backfill: create universities from distinct team.university strings
    if column_exists?(:teams, :university)
      MTeam.distinct.where.not(university: [nil, '']).pluck(:university).each do |name|
        uni = MUniversity.find_or_create_by!(name: name)
      end

      MTeam.where.not(university: [nil, '']).find_each do |t|
        uni = MUniversity.find_by(name: t.university)
        t.update_columns(university_id: uni&.id)
      end

      remove_column :teams, :university
    end
  end

  def down
    add_column :teams, :university, :string
    MTeam.reset_column_information
    if column_exists?(:teams, :university_id)
      MTeam.find_each do |t|
        if t.university_id
          uni = MUniversity.find_by(id: t.university_id)
          t.update_columns(university: uni&.name)
        end
      end
      remove_reference :teams, :university, foreign_key: true
    end
  end
end

