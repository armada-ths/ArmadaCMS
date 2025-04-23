export interface UserLogin {
  username: string;
  password: string;
}
export interface CreateUser extends UserLogin {
  name: string;
}
export interface User extends CreateUser {
  id: number;
}
export interface Tokens {
  refreshToken: string;
  accessToken: string;
}
