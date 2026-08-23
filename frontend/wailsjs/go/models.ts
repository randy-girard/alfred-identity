export namespace app {
	
	export class EqHostState {
	    current: string;
	    backup: string;
	    has_backup: boolean;
	
	    static createFrom(source: any = {}) {
	        return new EqHostState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.current = source["current"];
	        this.backup = source["backup"];
	        this.has_backup = source["has_backup"];
	    }
	}
	export class ImportAccountsResult {
	    message: string;
	    added: number;
	    updated: number;
	
	    static createFrom(source: any = {}) {
	        return new ImportAccountsResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.message = source["message"];
	        this.added = source["added"];
	        this.updated = source["updated"];
	    }
	}
	export class LocalAccountDTO {
	    name: string;
	    password: string;
	    aliases: string[];
	    has_password: boolean;
	    shared: boolean;
	    shared_user_ids: number[];
	    shared_role_ids: string[];
	    shared_group_ids: number[];
	    shared_sso_account_id: number;
	    in_use: boolean;
	    in_use_by?: string;
	    in_use_other: boolean;
	    last_login_at?: string;
	    last_login_by?: string;
	    last_login_other: boolean;
	
	    static createFrom(source: any = {}) {
	        return new LocalAccountDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.password = source["password"];
	        this.aliases = source["aliases"];
	        this.has_password = source["has_password"];
	        this.shared = source["shared"];
	        this.shared_user_ids = source["shared_user_ids"];
	        this.shared_role_ids = source["shared_role_ids"];
	        this.shared_group_ids = source["shared_group_ids"];
	        this.shared_sso_account_id = source["shared_sso_account_id"];
	        this.in_use = source["in_use"];
	        this.in_use_by = source["in_use_by"];
	        this.in_use_other = source["in_use_other"];
	        this.last_login_at = source["last_login_at"];
	        this.last_login_by = source["last_login_by"];
	        this.last_login_other = source["last_login_other"];
	    }
	}
	export class LocalCharacterDTO {
	    account: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new LocalCharacterDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.account = source["account"];
	        this.name = source["name"];
	    }
	}
	export class P99ProxyInstallDTO {
	    config_path: string;
	    config_dir: string;
	    accounts_csv: string;
	    characters_csv: string;
	    eq_directory?: string;
	    has_accounts: boolean;
	
	    static createFrom(source: any = {}) {
	        return new P99ProxyInstallDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.config_path = source["config_path"];
	        this.config_dir = source["config_dir"];
	        this.accounts_csv = source["accounts_csv"];
	        this.characters_csv = source["characters_csv"];
	        this.eq_directory = source["eq_directory"];
	        this.has_accounts = source["has_accounts"];
	    }
	}
	export class SourceDTO {
	    id: string;
	    name: string;
	    host: string;
	    notes?: string;
	    has_token: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SourceDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.host = source["host"];
	        this.notes = source["notes"];
	        this.has_token = source["has_token"];
	    }
	}
	export class SourceImportPreview {
	    name: string;
	    host: string;
	    notes?: string;
	    token?: string;
	
	    static createFrom(source: any = {}) {
	        return new SourceImportPreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.host = source["host"];
	        this.notes = source["notes"];
	        this.token = source["token"];
	    }
	}
	export class StatusDTO {
	    version: string;
	    connection_mode: string;
	    proxy_enabled: boolean;
	    sso_connected: boolean;
	    sso_is_admin: boolean;
	    sso_user_id: number;
	    active_source: string;
	    online: string[];
	    eq_directory: string;
	    listen: string;
	    sso_accounts: sso.AccountMeta[];
	    sso_online: sso.OnlineEntry[];
	    sso_directory: sso.DirectoryUser[];
	    sso_groups: sso.GroupDetail[];
	    sso_roles: sso.DiscordRole[];
	    sso_admin_users: sso.AdminUser[];
	    sso_admin_roles: sso.DiscordRole[];
	    share_activity: sso.ShareActivity;
	    sources: SourceDTO[];
	
	    static createFrom(source: any = {}) {
	        return new StatusDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.connection_mode = source["connection_mode"];
	        this.proxy_enabled = source["proxy_enabled"];
	        this.sso_connected = source["sso_connected"];
	        this.sso_is_admin = source["sso_is_admin"];
	        this.sso_user_id = source["sso_user_id"];
	        this.active_source = source["active_source"];
	        this.online = source["online"];
	        this.eq_directory = source["eq_directory"];
	        this.listen = source["listen"];
	        this.sso_accounts = this.convertValues(source["sso_accounts"], sso.AccountMeta);
	        this.sso_online = this.convertValues(source["sso_online"], sso.OnlineEntry);
	        this.sso_directory = this.convertValues(source["sso_directory"], sso.DirectoryUser);
	        this.sso_groups = this.convertValues(source["sso_groups"], sso.GroupDetail);
	        this.sso_roles = this.convertValues(source["sso_roles"], sso.DiscordRole);
	        this.sso_admin_users = this.convertValues(source["sso_admin_users"], sso.AdminUser);
	        this.sso_admin_roles = this.convertValues(source["sso_admin_roles"], sso.DiscordRole);
	        this.share_activity = this.convertValues(source["share_activity"], sso.ShareActivity);
	        this.sources = this.convertValues(source["sources"], SourceDTO);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class UpdateInfo {
	    update_available: boolean;
	    current: string;
	    latest: string;
	    release_url: string;
	    asset_name?: string;
	    asset_url?: string;
	    can_apply: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.update_available = source["update_available"];
	        this.current = source["current"];
	        this.latest = source["latest"];
	        this.release_url = source["release_url"];
	        this.asset_name = source["asset_name"];
	        this.asset_url = source["asset_url"];
	        this.can_apply = source["can_apply"];
	        this.error = source["error"];
	    }
	}

}

export namespace keys {
	
	export class Accelerator {
	    Key: string;
	    Modifiers: string[];
	
	    static createFrom(source: any = {}) {
	        return new Accelerator(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Key = source["Key"];
	        this.Modifiers = source["Modifiers"];
	    }
	}

}

export namespace logbuf {
	
	export class Entry {
	    time: string;
	    level: string;
	    message: string;
	    attrs?: string;
	
	    static createFrom(source: any = {}) {
	        return new Entry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.time = source["time"];
	        this.level = source["level"];
	        this.message = source["message"];
	        this.attrs = source["attrs"];
	    }
	}

}

export namespace menu {
	
	export class MenuItem {
	    Label: string;
	    Role: number;
	    Accelerator?: keys.Accelerator;
	    Type: string;
	    Disabled: boolean;
	    Hidden: boolean;
	    Checked: boolean;
	    SubMenu?: Menu;
	
	    static createFrom(source: any = {}) {
	        return new MenuItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Label = source["Label"];
	        this.Role = source["Role"];
	        this.Accelerator = this.convertValues(source["Accelerator"], keys.Accelerator);
	        this.Type = source["Type"];
	        this.Disabled = source["Disabled"];
	        this.Hidden = source["Hidden"];
	        this.Checked = source["Checked"];
	        this.SubMenu = this.convertValues(source["SubMenu"], Menu);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Menu {
	    Items: MenuItem[];
	
	    static createFrom(source: any = {}) {
	        return new Menu(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Items = this.convertValues(source["Items"], MenuItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace options {
	
	export class SecondInstanceData {
	    Args: string[];
	    WorkingDirectory: string;
	
	    static createFrom(source: any = {}) {
	        return new SecondInstanceData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Args = source["Args"];
	        this.WorkingDirectory = source["WorkingDirectory"];
	    }
	}

}

export namespace sources {
	
	export class Source {
	    id: string;
	    name: string;
	    host: string;
	    token: string;
	    notes?: string;
	    url?: string;
	
	    static createFrom(source: any = {}) {
	        return new Source(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.host = source["host"];
	        this.token = source["token"];
	        this.notes = source["notes"];
	        this.url = source["url"];
	    }
	}

}

export namespace sso {
	
	export class AccountMeta {
	    id: number;
	    username: string;
	    disabled: boolean;
	    elevated: boolean;
	    required_role_id: string;
	    required_role_ids: string[];
	    required_user_id: number;
	    group_ids: number[];
	    restricted: boolean;
	    owner_user_id: number;
	    shared_user_ids: number[];
	    aliases: string[];
	    tags: string[];
	    characters: string[];
	
	    static createFrom(source: any = {}) {
	        return new AccountMeta(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.username = source["username"];
	        this.disabled = source["disabled"];
	        this.elevated = source["elevated"];
	        this.required_role_id = source["required_role_id"];
	        this.required_role_ids = source["required_role_ids"];
	        this.required_user_id = source["required_user_id"];
	        this.group_ids = source["group_ids"];
	        this.restricted = source["restricted"];
	        this.owner_user_id = source["owner_user_id"];
	        this.shared_user_ids = source["shared_user_ids"];
	        this.aliases = source["aliases"];
	        this.tags = source["tags"];
	        this.characters = source["characters"];
	    }
	}
	export class AdminUser {
	    id: number;
	    discord_id: string;
	    display_name: string;
	    role_ids: string[];
	    access_revoked: boolean;
	    has_active_token: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AdminUser(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.discord_id = source["discord_id"];
	        this.display_name = source["display_name"];
	        this.role_ids = source["role_ids"];
	        this.access_revoked = source["access_revoked"];
	        this.has_active_token = source["has_active_token"];
	    }
	}
	export class DirectoryUser {
	    id: number;
	    discord_id: string;
	    display_name: string;
	
	    static createFrom(source: any = {}) {
	        return new DirectoryUser(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.discord_id = source["discord_id"];
	        this.display_name = source["display_name"];
	    }
	}
	export class DiscordRole {
	    id: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new DiscordRole(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	    }
	}
	export class GroupUser {
	    id: number;
	    discord_id: string;
	    display_name: string;
	
	    static createFrom(source: any = {}) {
	        return new GroupUser(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.discord_id = source["discord_id"];
	        this.display_name = source["display_name"];
	    }
	}
	export class GroupDetail {
	    id: number;
	    name: string;
	    description: string;
	    web_role: string;
	    users: GroupUser[];
	    user_ids: number[];
	    role_ids: string[];
	    account_ids: number[];
	
	    static createFrom(source: any = {}) {
	        return new GroupDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.web_role = source["web_role"];
	        this.users = this.convertValues(source["users"], GroupUser);
	        this.user_ids = source["user_ids"];
	        this.role_ids = source["role_ids"];
	        this.account_ids = source["account_ids"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class OnlineEntry {
	    account_id: number;
	    character_name: string;
	
	    static createFrom(source: any = {}) {
	        return new OnlineEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.account_id = source["account_id"];
	        this.character_name = source["character_name"];
	    }
	}
	export class ShareOnlineEntry {
	    account_id: number;
	    account_username: string;
	    character_name: string;
	    user_id: number;
	    user_display_name: string;
	    user_discord_id: string;
	    actor_is_owner: boolean;
	    // Go type: time
	    last_seen: any;
	
	    static createFrom(source: any = {}) {
	        return new ShareOnlineEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.account_id = source["account_id"];
	        this.account_username = source["account_username"];
	        this.character_name = source["character_name"];
	        this.user_id = source["user_id"];
	        this.user_display_name = source["user_display_name"];
	        this.user_discord_id = source["user_discord_id"];
	        this.actor_is_owner = source["actor_is_owner"];
	        this.last_seen = this.convertValues(source["last_seen"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ShareLoginEntry {
	    id: number;
	    // Go type: time
	    created_at: any;
	    user_id: number;
	    actor_name: string;
	    actor_discord_id: string;
	    account_id: number;
	    account_username: string;
	    detail: string;
	    actor_is_owner: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ShareLoginEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.user_id = source["user_id"];
	        this.actor_name = source["actor_name"];
	        this.actor_discord_id = source["actor_discord_id"];
	        this.account_id = source["account_id"];
	        this.account_username = source["account_username"];
	        this.detail = source["detail"];
	        this.actor_is_owner = source["actor_is_owner"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ShareActivity {
	    logins: ShareLoginEntry[];
	    online: ShareOnlineEntry[];
	
	    static createFrom(source: any = {}) {
	        return new ShareActivity(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.logins = this.convertValues(source["logins"], ShareLoginEntry);
	        this.online = this.convertValues(source["online"], ShareOnlineEntry);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	

}

