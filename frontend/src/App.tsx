import { ReactNode } from "react";
import { Box, Stack } from "@mui/material";
import {
  Admin,
  AppBar,
  DashboardMenuItem,
  Layout,
  Logout,
  Menu,
  Resource,
  TitlePortal,
  UserMenu,
} from "react-admin";
import { AuditLogList } from "./components/AuditLog/AuditLogList";
import { AuditLogShow } from "./components/AuditLog/AuditLogShow";
import { BlogpostCreate } from "./components/Blogpost/BlogpostCreate";
import { BlogpostEdit } from "./components/Blogpost/BlogpostEdit";
import { BlogpostList } from "./components/Blogpost/BlogpostList";
import { ChangePasswordButton } from "./components/ChangePasswordButton";
import { CustomLoginPage } from "./components/CustomLoginPage";
import { Dashboard } from "./components/Dashboard/Dashboard";
import { EmploymentCreate } from "./components/Employment/EmploymentCreate";
import { EmploymentEdit } from "./components/Employment/EmploymentEdit";
import { EmploymentList } from "./components/Employment/EmploymentList";
import { EventCreate } from "./components/Event/EventCreate";
import { EventEdit } from "./components/Event/EventEdit";
import { EventList } from "./components/Event/EventList";
import { ExhibitorCreate } from "./components/Exhibitor/ExhibitorCreate";
import { ExhibitorEdit } from "./components/Exhibitor/ExhibitorEdit";
import { ExhibitorList } from "./components/Exhibitor/ExhibitorList";
import { FairDateCreate } from "./components/FairDate/FairDateCreate";
import { FairDateEdit } from "./components/FairDate/FairDateEdit";
import { FairDateList } from "./components/FairDate/FairDateList";
import { FeatureFlagEdit } from "./components/FeatureFlag/FeatureFlagEdit";
import { FeatureFlagList } from "./components/FeatureFlag/FeatureFlagList";
import { HighlightCardCreate } from "./components/HighlightCard/HighlightCardCreate";
import { HighlightCardEdit } from "./components/HighlightCard/HighlightCardEdit";
import { HighlightCardList } from "./components/HighlightCard/HighlightCardList";
import { IndustryCreate } from "./components/Industry/IndustryCreate";
import { IndustryEdit } from "./components/Industry/IndustryEdit";
import { IndustryList } from "./components/Industry/IndustryList";
import { ProfileCreate } from "./components/Profile/ProfileCreate";
import { ProfileEdit } from "./components/Profile/ProfileEdit";
import { ProfileList } from "./components/Profile/ProfileList";
import { ProgramCreate } from "./components/Program/ProgramCreate";
import { ProgramEdit } from "./components/Program/ProgramEdit";
import { ProgramList } from "./components/Program/ProgramList";
import { RecruitmentPeriodCreate } from "./components/RecruitmentPeriod/RecruitmentPeriodCreate";
import { RecruitmentPeriodEdit } from "./components/RecruitmentPeriod/RecruitmentPeriodEdit";
import { RecruitmentPeriodList } from "./components/RecruitmentPeriod/RecruitmentPeriodList";
import { RecruitmentRoleCreate } from "./components/RecruitmentRole/RecruitmentRoleCreate";
import { RecruitmentRoleEdit } from "./components/RecruitmentRole/RecruitmentRoleEdit";
import { RecruitmentRoleList } from "./components/RecruitmentRole/RecruitmentRoleList";
import { RoleCreate } from "./components/Role/RoleCreate";
import { RoleEdit } from "./components/Role/RoleEdit";
import { RoleList } from "./components/Role/RoleList";
import { TeamCreate } from "./components/Team/TeamCreate";
import { TeamEdit } from "./components/Team/TeamEdit";
import { TeamList } from "./components/Team/TeamList";
import { UserCreate } from "./components/User/UserCreate";
import { UserEdit } from "./components/User/UserEdit";
import { UserList } from "./components/User/UserList";
import { ArmadaBrandLockup } from "./components/Branding";
import { authProvider } from "./context/authProvider";
import { dataProvider } from "./dataProvider";
import { armadaDarkTheme, armadaTheme } from "./theme";

export const MyMenu = () => (
  <Menu
    sx={{
      "& .RaMenuItemLink-root": {
        borderRadius: 1,
        marginBlock: 0,
        marginInline: 0.5,
      },
      "& .RaMenuItemLink-active": {
        backgroundColor: "primary.main",
        color: "primary.contrastText",
        fontWeight: 700,
      },
      "& .RaMenuItemLink-icon": {
        minWidth: 36,
      },
    }}
  >
    <DashboardMenuItem primaryText="Dashboard" />
    <Menu.ResourceItems />
  </Menu>
);

const MyUserMenu = () => (
  <UserMenu>
    <ChangePasswordButton />
    <Logout />
  </UserMenu>
);

const MyAppBar = () => (
  <AppBar
    userMenu={<MyUserMenu />}
    sx={{
      "& .RaAppBar-toolbar": {
        gap: 2,
      },
    }}
  >
    <Stack
      direction="row"
      alignItems="center"
      spacing={2}
      sx={{ flex: 1, minWidth: 0 }}
    >
      <ArmadaBrandLockup compact />
      <Box
        sx={{
          minWidth: 0,
          flex: 1,
          display: "flex",
          alignItems: "center",
          justifyContent: { xs: "flex-end", md: "center" },
          "& .RaAppBar-title": {
            overflow: "hidden",
            textOverflow: "ellipsis",
            whiteSpace: "nowrap",
            color: "rgba(255, 255, 255, 0.82)",
          },
          "& .RaAppBar-title h6, & .RaAppBar-title span": {
            fontFamily: '"Bebas Neue", "Oswald", "Arial Narrow", sans-serif',
            fontSize: { xs: "1rem", md: "1.15rem" },
            letterSpacing: "0.08em",
          },
        }}
      >
        <TitlePortal />
      </Box>
    </Stack>
  </AppBar>
);

export const MyLayout = ({ children }: { children?: ReactNode }) => (
  <Layout
    menu={MyMenu}
    appBar={MyAppBar}
    sx={{
      "& .RaLayout-content": {
        backgroundColor: "transparent",
      },
      "& .RaLayout-contentWithSidebar": {
        backgroundColor: "transparent",
      },
    }}
  >
    {children}
  </Layout>
);

export const App = () => (
  <Admin
    basename="/admin"
    dataProvider={dataProvider}
    authProvider={authProvider}
    layout={MyLayout}
    loginPage={CustomLoginPage}
    dashboard={Dashboard}
    theme={armadaTheme}
    darkTheme={armadaDarkTheme}
  >
    <Resource
      name="customusers"
      options={{ label: "Custom users" }}
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
      name="programs"
      list={ProgramList}
      create={ProgramCreate}
      edit={ProgramEdit}
    />
    <Resource
      name="industries"
      list={IndustryList}
      create={IndustryCreate}
      edit={IndustryEdit}
    />
    <Resource
      name="events"
      list={EventList}
      create={EventCreate}
      edit={EventEdit}
    />
    <Resource
      name="exhibitors"
      list={ExhibitorList}
      create={ExhibitorCreate}
      edit={ExhibitorEdit}
    />
    <Resource
      name="employments"
      list={EmploymentList}
      create={EmploymentCreate}
      edit={EmploymentEdit}
    />
    <Resource
      name="fairdates"
      options={{ label: "Fair dates" }}
      list={FairDateList}
      create={FairDateCreate}
      edit={FairDateEdit}
    />
    <Resource
      name="featureflags"
      options={{ label: "Feature flags" }}
      list={FeatureFlagList}
      edit={FeatureFlagEdit}
    />

    <Resource
      name="highlightcards"
      options={{ label: "Highlight cards" }}
      list={HighlightCardList}
      create={HighlightCardCreate}
      edit={HighlightCardEdit}
    />

    <Resource
      name="roles"
      list={RoleList}
      create={RoleCreate}
      edit={RoleEdit}
    />

    <Resource
      name="recruitmentperiods"
      options={{ label: "Recruitment periods" }}
      list={RecruitmentPeriodList}
      create={RecruitmentPeriodCreate}
      edit={RecruitmentPeriodEdit}
    />

    <Resource
      name="recruitmentroles"
      options={{ label: "Recruitment roles" }}
      list={RecruitmentRoleList}
      create={RecruitmentRoleCreate}
      edit={RecruitmentRoleEdit}
    />

    <Resource
      name="auditlogs"
      options={{ label: "Audit logs" }}
      list={AuditLogList}
      show={AuditLogShow}
    />

    <Resource
      name="blogposts"
      options={{ label: "Blog posts" }}
      list={BlogpostList}
      create={BlogpostCreate}
      edit={BlogpostEdit}
    />
  </Admin>
);
