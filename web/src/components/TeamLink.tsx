import { Link } from "react-router-dom";

/** Renders a team name as a hyperlink to the team detail page. */
export function TeamLink({ id, name }: { id: number; name: string }) {
  return <Link to={`/teams/${id}`}>{name}</Link>;
}
