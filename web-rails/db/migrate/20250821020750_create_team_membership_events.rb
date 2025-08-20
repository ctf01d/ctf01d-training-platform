class CreateTeamMembershipEvents < ActiveRecord::Migration[8.0]
  def change
    create_table :team_membership_events do |t|
      t.references :team, null: false, foreign_key: true
      t.references :user, null: false, foreign_key: true # целевой пользователь (участник)
      t.integer :actor_id # кто совершил действие (user.id)
      t.string :action, null: false
      t.string :from_role
      t.string :to_role
      t.string :from_status
      t.string :to_status

      t.timestamps
    end

    add_index :team_membership_events, :actor_id
    add_index :team_membership_events, [:team_id, :created_at]
  end
end

