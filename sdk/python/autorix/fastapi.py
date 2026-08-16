import typing
from .client import AutorixClient, User

class AutorixSecurity:
    def __init__(self, client: typing.Optional[AutorixClient] = None):
        self.client = client or AutorixClient()

    def get_current_user(
        self,
        x_user_id: typing.Optional[str] = None,
        x_user_email: typing.Optional[str] = None,
        x_user_roles: typing.Optional[str] = None,
    ) -> typing.Optional[User]:
        if not x_user_id:
            return None

        roles = [r.strip() for r in x_user_roles.split(",")] if x_user_roles else []
        return User(id=x_user_id, email=x_user_email, roles=roles)
