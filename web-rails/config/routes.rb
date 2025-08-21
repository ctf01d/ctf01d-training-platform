Rails.application.routes.draw do
  root to: 'services#index'

  resource :session, only: %i[new create destroy]

  resources :universities
  resources :results
  resources :games do
    member do
      get :manage_services
      post :add_service
      delete :remove_service
      post :finalize
      delete :unfinalize
    end
  end
  resources :services do
    member do
      post :toggle_public
      post :check_checker
    end
  end
  resources :team_memberships
  resources :team_memberships do
    member do
      post :approve
      post :reject
      post :accept
      post :decline
      post :set_role
    end
  end
  resources :teams do
    member do
      post :join_request
      post :invite
    end
  end
  resources :users
  get 'scoreboard', to: 'scoreboards#index', as: :scoreboard
  resource :profile, only: %i[show edit update]
  # Define your application routes per the DSL in https://guides.rubyonrails.org/routing.html

  # Reveal health status on /up that returns 200 if the app boots with no exceptions, otherwise 500.
  # Can be used by load balancers and uptime monitors to verify that the app is live.
  get "up" => "rails/health#show", as: :rails_health_check

  # Render dynamic PWA files from app/views/pwa/* (remember to link manifest in application.html.erb)
  # get "manifest" => "rails/pwa#manifest", as: :pwa_manifest
  # get "service-worker" => "rails/pwa#service_worker", as: :pwa_service_worker

  # Defines the root path route ("/")
  # root "posts#index"
end
