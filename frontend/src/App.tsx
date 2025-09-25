import { Admin, Resource } from "react-admin";
import { Layout } from "./Layout";
import { dataProvider } from "./dataProvider";
import { UserList } from "./components/User/UserList";
import { UserCreate } from "./components/User/UserCreate";
import { UserEdit } from "./components/User/UserEdit";
import { ProfileList } from "./components/Profile/ProfileList";
import { ProfileCreate } from "./components/Profile/ProfileCreate";
import { ProfileEdit } from "./components/Profile/ProfileEdit";
import { TeamList } from "./components/Team/TeamList";
import { TeamCreate } from "./components/Team/TeamCreate";
import { TeamEdit } from "./components/Team/TeamEdit";
import { authProvider } from "./context/authProvider";
import { TimelineList } from "./components/Timeline/TimelineList";
import { TimelineCreate } from "./components/Timeline/TimelineCreate";
import { TimelineEdit } from "./components/Timeline/TimelineEdit";

export const App = () => (
  <Admin
    dataProvider={dataProvider}
    authProvider={authProvider}
    layout={Layout}
  >
    <Resource
      name="customusers"
      list={UserList}
      create={UserCreate}
      edit={UserEdit}
    />
    <Resource
      name="profiles"
      list={ProfileList}
      create={ProfileCreate}
      edit={ProfileEdit}
    />
    <Resource
      name="teams"
      list={TeamList}
      create={TeamCreate}
      edit={TeamEdit}
    />
    <Resource
      name="timeline"
      list={TimelineList}
      create={TimelineCreate}
      edit={TimelineEdit}
    />
  </Admin>
);
