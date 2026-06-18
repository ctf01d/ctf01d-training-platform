import { createBrowserRouter } from "react-router-dom";
import Layout from "./components/Layout";
import { ProtectedRoute, AdminRoute } from "./components/ProtectedRoute";
import LoginPage from "./pages/LoginPage";
import GamesPage from "./pages/GamesPage";
import GameDetailPage from "./pages/GameDetailPage";
import ServicesPage from "./pages/ServicesPage";
import ServiceDetailPage from "./pages/ServiceDetailPage";
import TeamsPage from "./pages/TeamsPage";
import TeamDetailPage from "./pages/TeamDetailPage";
import UniversitiesPage from "./pages/UniversitiesPage";
import UniversityDetailPage from "./pages/UniversityDetailPage";
import UsersPage from "./pages/UsersPage";
import ResultsPage from "./pages/ResultsPage";
import ProfilePage from "./pages/ProfilePage";
import ScoreboardPage from "./pages/ScoreboardPage";

export const router = createBrowserRouter([
  {
    path: "/login",
    element: <LoginPage />,
  },
  {
    element: <Layout />,
    children: [
      { index: true, element: <GamesPage /> },
      { path: "games", element: <GamesPage /> },
      { path: "games/:id", element: <GameDetailPage /> },
      { path: "services", element: <ServicesPage /> },
      { path: "services/:id", element: <ServiceDetailPage /> },
      { path: "teams", element: <TeamsPage /> },
      { path: "teams/:id", element: <TeamDetailPage /> },
      { path: "scoreboard", element: <ScoreboardPage /> },
      {
        path: "profile",
        element: (
          <ProtectedRoute>
            <ProfilePage />
          </ProtectedRoute>
        ),
      },
      {
        path: "results",
        element: (
          <ProtectedRoute>
            <ResultsPage />
          </ProtectedRoute>
        ),
      },
      {
        path: "universities",
        element: (
          <AdminRoute>
            <UniversitiesPage />
          </AdminRoute>
        ),
      },
      {
        path: "universities/:id",
        element: (
          <AdminRoute>
            <UniversityDetailPage />
          </AdminRoute>
        ),
      },
      {
        path: "users",
        element: (
          <AdminRoute>
            <UsersPage />
          </AdminRoute>
        ),
      },
    ],
  },
]);
