import type { User, UserProfileUpdate } from "../api/users";

export type UserProfileFormState = {
  display_name: string;
  bio: string;
  telegram: string;
  github: string;
  email: string;
};

export type UserPasswordFormState = {
  password: string;
  confirm: string;
};

export function emptyUserProfileForm(): UserProfileFormState {
  return {
    display_name: "",
    bio: "",
    telegram: "",
    github: "",
    email: "",
  };
}

export function userProfileFormFromUser(user: User): UserProfileFormState {
  return {
    display_name: user.display_name ?? "",
    bio: user.bio ?? "",
    telegram: user.telegram ?? "",
    github: user.github ?? "",
    email: user.email ?? "",
  };
}

export function profileUpdateFromForm(
  form: UserProfileFormState,
): UserProfileUpdate {
  return {
    display_name: form.display_name,
    bio: form.bio || null,
    telegram: form.telegram || null,
    github: form.github || null,
    email: form.email || null,
  };
}

export function userTitle(user: User): string {
  return user.display_name || user.user_name;
}
