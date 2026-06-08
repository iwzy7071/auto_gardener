#!/usr/bin/env python3
import argparse, base64, crypt, json, os, re, secrets, shlex, shutil, socket, string, subprocess, sys, time
from pathlib import Path

ROOT = Path('/etc/gardener-relay')
USERS = ROOT / 'users'
STATE = ROOT / 'instances.json'
NGINX_DIR = Path('/etc/nginx/conf.d')
FRPS_CONF = Path('/etc/frp/frps.toml')
DOWNLOAD_ROOT = Path('/srv/gardener-downloads/public')
PROVISION_ROOT = DOWNLOAD_ROOT / 'provision'
SERVER_ADDR = os.environ.get('GARDENER_RELAY_SERVER_ADDR', 'YOUR_RELAY_SERVER')
FRPS_PORT = int(os.environ.get('GARDENER_RELAY_FRPS_PORT', '27000'))
PUBLIC_START = int(os.environ.get('GARDENER_RELAY_PUBLIC_START', '28081'))
PUBLIC_END = int(os.environ.get('GARDENER_RELAY_PUBLIC_END', '28100'))
REMOTE_START = int(os.environ.get('GARDENER_RELAY_REMOTE_START', '18081'))
REMOTE_END = int(os.environ.get('GARDENER_RELAY_REMOTE_END', '18100'))
PROVISION_RETENTION_SECONDS = int(os.environ.get('GARDENER_RELAY_PROVISION_RETENTION_SECONDS', '86400'))
RELAY_PUBLIC_BASE_URL = os.environ.get('GARDENER_RELAY_PUBLIC_BASE_URL', f'http://{SERVER_ADDR}')
PACKAGE_URL = os.environ.get('GARDENER_RELAY_WINDOWS_PACKAGE_URL', f'{RELAY_PUBLIC_BASE_URL}/downloads/Gardener-Windows.zip')
INSTALL_SCRIPT_URL = os.environ.get('GARDENER_RELAY_WINDOWS_INSTALL_SCRIPT_URL', f'{RELAY_PUBLIC_BASE_URL}/downloads/install-gardener.ps1')
MAC_INSTALL_SCRIPT_URL = os.environ.get('GARDENER_RELAY_MAC_INSTALL_SCRIPT_URL', f'{RELAY_PUBLIC_BASE_URL}/downloads/install-gardener-macos.sh')
MAC_PACKAGE_URLS = {
    'arm64': os.environ.get('GARDENER_RELAY_MAC_ARM64_PACKAGE_URL', f'{RELAY_PUBLIC_BASE_URL}/downloads/Gardener-macOS-arm64.tar.gz'),
    'amd64': os.environ.get('GARDENER_RELAY_MAC_AMD64_PACKAGE_URL', f'{RELAY_PUBLIC_BASE_URL}/downloads/Gardener-macOS-amd64.tar.gz'),
}


def require_relay_configured():
    placeholders = {'', 'YOUR_RELAY_SERVER', 'YOUR_VPS_IP'}
    if SERVER_ADDR in placeholders or 'example.com' in SERVER_ADDR:
        raise SystemExit('error: relay server address is not configured. Set GARDENER_RELAY_SERVER_ADDR and GARDENER_RELAY_PUBLIC_BASE_URL from config/gardener-relay.env.local')
    if RELAY_PUBLIC_BASE_URL.endswith('YOUR_RELAY_SERVER') or 'YOUR_RELAY_SERVER' in RELAY_PUBLIC_BASE_URL or 'example.com' in RELAY_PUBLIC_BASE_URL:
        raise SystemExit('error: relay public base URL is not configured. Set GARDENER_RELAY_PUBLIC_BASE_URL from config/gardener-relay.env.local')

def run(cmd, check=True):
    return subprocess.run(cmd, check=check, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)


def load_state():
    if not STATE.exists():
        return {'instances': []}
    try:
        return json.loads(STATE.read_text())
    except (json.JSONDecodeError, UnicodeDecodeError) as exc:
        raise SystemExit(f'error: relay state is malformed: {exc}') from exc


def save_state(data):
    ROOT.mkdir(parents=True, exist_ok=True)
    tmp = STATE.with_suffix('.tmp')
    tmp.write_text(json.dumps(data, ensure_ascii=False, indent=2) + '\n')
    os.chmod(tmp, 0o600)
    tmp.replace(STATE)
    os.chmod(STATE, 0o600)


def sanitize_user(user):
    user = user.strip().lower()
    allowed = set(string.ascii_lowercase + string.digits + '-_.')
    if not user or any(c not in allowed for c in user):
        raise SystemExit('error: user must contain only a-z, 0-9, dot, - or _')
    if user in ('.', '..') or '..' in user or user.startswith('.') or user.endswith('.'):
        raise SystemExit('error: invalid dot placement in user name')
    if len(user) > 48:
        raise SystemExit('error: user name too long')
    return user


def relay_id_for_user(user):
    safe = re.sub(r'[^a-z0-9_-]+', '-', user).strip('-_')
    return safe or 'user'


def sanitize_setup_key(key):
    key = key.strip()
    allowed = set(string.ascii_letters + string.digits + '-_')
    if not key or any(c not in allowed for c in key):
        raise SystemExit('error: setup key must contain only a-z, A-Z, 0-9, - or _')
    if len(key) < 20:
        raise SystemExit('error: setup key too short')
    return key


def make_setup_key():
    return 'sk_' + secrets.token_urlsafe(24).rstrip('=')

def setup_key_from_args(args):
    if args.setup_key:
        return args.setup_key
    if getattr(args, 'setup_key_file', None):
        try:
            return Path(args.setup_key_file).read_text().strip()
        except OSError as exc:
            raise SystemExit(f'error: cannot read setup key file: {exc}') from exc
    return os.environ.get('GARDENER_RELAY_SETUP_KEY', '')


def validate_tcp_port(port, label):
    if int(port) < 1 or int(port) > 65535:
        raise SystemExit(f'error: {label} must be between 1 and 65535')
    return int(port)

def port_listening(port, host='127.0.0.1'):
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.settimeout(0.25)
    try:
        return s.connect_ex((host, port)) == 0
    finally:
        s.close()


def find_free_slot(instances):
    used_public = {int(i['publicPort']) for i in instances}
    used_remote = {int(i['remotePort']) for i in instances}
    for idx, public in enumerate(range(PUBLIC_START, PUBLIC_END + 1)):
        remote = REMOTE_START + idx
        if public in used_public or remote in used_remote:
            continue
        if port_listening(public, '0.0.0.0') or port_listening(remote, '127.0.0.1'):
            continue
        return public, remote
    raise SystemExit('error: no free port slot available')


def read_frp_token():
    text = FRPS_CONF.read_text()
    for line in text.splitlines():
        line = line.strip()
        if line.startswith('auth.token'):
            return line.split('=', 1)[1].strip().strip('"')
    raise SystemExit('error: cannot find auth.token in /etc/frp/frps.toml')


def make_password():
    alphabet = string.ascii_letters + string.digits
    return ''.join(secrets.choice(alphabet) for _ in range(18))


def password_from_args(args):
    if getattr(args, 'password_file', None):
        password = Path(args.password_file).read_text().strip()
        if not password:
            raise SystemExit('error: password file is empty')
        return password
    if getattr(args, 'password', None):
        return args.password
    env_password = os.environ.get('GARDENER_RELAY_PASSWORD', '').strip()
    if env_password:
        return env_password
    return make_password()


def htpasswd_hash(password):
    salt = '$6$' + secrets.token_urlsafe(12)
    return crypt.crypt(password, salt)


def nginx_conf(user, public_port, remote_port):
    return f'''server {{
    listen {public_port};
    server_name _;

    client_max_body_size 200m;
    client_header_timeout 10s;
    client_body_timeout 30s;
    add_header Cache-Control "no-store" always;
    add_header X-Content-Type-Options "nosniff" always;

    auth_basic "Gardener";
    auth_basic_user_file /etc/gardener-relay/users/{user}/htpasswd;

    location / {{
        proxy_pass http://127.0.0.1:{remote_port};
        proxy_http_version 1.1;
        proxy_set_header Host $http_host;
        proxy_set_header X-Forwarded-Host $http_host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Authorization "";
        proxy_set_header Cookie "";
        proxy_buffering off;
        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;
    }}
}}
'''


def toml_string(value):
    return json.dumps(str(value))


def frpc_conf(user, remote_port):
    token = read_frp_token()
    proxy_name = 'gardener-' + relay_id_for_user(user)
    return f'''serverAddr = {toml_string(SERVER_ADDR)}
serverPort = {FRPS_PORT}

auth.method = "token"
auth.token = {toml_string(token)}

[[proxies]]
name = {toml_string(proxy_name)}
type = "tcp"
localIP = "127.0.0.1"
localPort = 8080
remotePort = {remote_port}
'''


def set_nginx_readable(path):
    try:
        import grp
        gid = None
        for group in ('nginx', 'www-data'):
            try:
                gid = grp.getgrnam(group).gr_gid
                break
            except KeyError:
                continue
        if gid is not None:
            os.chown(path, 0, gid)
    except Exception:
        pass


def ensure_permissions(user_dir=None):
    ROOT.mkdir(parents=True, exist_ok=True)
    USERS.mkdir(parents=True, exist_ok=True)
    for path in [ROOT, USERS]:
        set_nginx_readable(path)
        os.chmod(path, 0o750)
    if user_dir is not None:
        set_nginx_readable(user_dir)
        os.chmod(user_dir, 0o750)


def cleanup_old_provisions(now=None):
    if PROVISION_RETENTION_SECONDS <= 0 or not PROVISION_ROOT.exists():
        return 0
    now = time.time() if now is None else now
    removed = 0
    for entry in PROVISION_ROOT.iterdir():
        if not entry.is_dir():
            continue
        provision_file = entry / 'gardener.provision.json'
        try:
            mtime = provision_file.stat().st_mtime if provision_file.exists() else entry.stat().st_mtime
        except OSError:
            continue
        if now - mtime > PROVISION_RETENTION_SECONDS:
            shutil.rmtree(entry, ignore_errors=True)
            removed += 1
    return removed


def gc_provisions(args):
    removed = cleanup_old_provisions()
    print(json.dumps({'removed': removed}, ensure_ascii=False))


def write_provision(instance, password, setup_key=None):
    cleanup_old_provisions()
    setup_key = sanitize_setup_key(setup_key) if setup_key else make_setup_key()
    provision_dir = PROVISION_ROOT / setup_key
    if provision_dir.exists():
        raise SystemExit('error: setup key already exists; please choose another key')
    provision_dir.mkdir(parents=True, exist_ok=False)
    provision = {
        'schemaVersion': 1,
        'user': instance['user'],
        'publicUrl': instance['url'],
        'webUsername': instance['basicAuthUser'],
        'webPassword': password,
        'packageUrl': PACKAGE_URL,
        'installScriptUrl': INSTALL_SCRIPT_URL,
        'macInstallScriptUrl': MAC_INSTALL_SCRIPT_URL,
        'macPackageUrls': MAC_PACKAGE_URLS,
        'frpcToml': (USERS / instance['user'] / 'frpc.toml').read_text(),
        'createdAt': time.strftime('%Y-%m-%dT%H:%M:%S%z'),
        'note': 'Treat this setup key as a secret. Anyone with this URL can configure this Gardener relay client.',
    }
    path = provision_dir / 'gardener.provision.json'
    path.write_text(json.dumps(provision, ensure_ascii=False, indent=2) + '\n')
    set_nginx_readable(provision_dir)
    set_nginx_readable(path)
    os.chmod(provision_dir, 0o750)
    os.chmod(path, 0o640)
    return setup_key, path


def powershell_single_quote(value):
    return "'" + str(value).replace("'", "''") + "'"

def install_command(setup_key):
    script_url = powershell_single_quote(INSTALL_SCRIPT_URL)
    relay_base_url = powershell_single_quote(RELAY_PUBLIC_BASE_URL)
    safe_setup_key = powershell_single_quote(setup_key)
    return ('powershell -ExecutionPolicy Bypass -Command '
            f'"iwr {script_url} -OutFile install-gardener.ps1; '
            f'.\\install-gardener.ps1 -RelayBaseUrl {relay_base_url} -SetupKey {safe_setup_key} -DesktopShortcut -StartMenuShortcut -StartAfterInstall"')


def mac_install_command(setup_key):
    script_url = shlex.quote(MAC_INSTALL_SCRIPT_URL)
    relay_base_url = shlex.quote(RELAY_PUBLIC_BASE_URL)
    safe_setup_key = shlex.quote(setup_key)
    return f'curl -fsSL {script_url} -o install-gardener-macos.sh && bash install-gardener-macos.sh --relay-base-url {relay_base_url} --setup-key {safe_setup_key}'


def reload_nginx():
    test = run(['nginx', '-t'], check=False)
    if test.returncode != 0:
        raise SystemExit(test.stderr + test.stdout)
    run(['systemctl', 'reload', 'nginx'])


def redact_add_result(result):
    redacted = dict(result)
    for key in ['password', 'setupKey', 'provisionUrl', 'provisionPath']:
        if redacted.get(key):
            redacted[key] = '<redacted>'
    redacted.pop('installCommand', None)
    redacted.pop('macInstallCommand', None)
    redacted['secretsRedacted'] = True
    redacted['nextStep'] = 'Run show --with-provision for explicit secret retrieval if needed.'
    return redacted


def add_user(args):
    require_relay_configured()
    user = sanitize_user(args.user)
    data = load_state()
    if any(i['user'] == user for i in data['instances']):
        raise SystemExit(f'error: user {user} already exists')
    proxy_name = f'gardener-{relay_id_for_user(user)}'
    if any(i.get('proxyName') == proxy_name for i in data['instances']):
        raise SystemExit(f'error: proxy name {proxy_name} already exists; choose another user name')
    if bool(args.public_port) != bool(args.remote_port):
        raise SystemExit('error: --public-port and --remote-port must be provided together')
    public, remote = (args.public_port, args.remote_port) if args.public_port and args.remote_port else find_free_slot(data['instances'])
    public = validate_tcp_port(public, 'public port')
    remote = validate_tcp_port(remote, 'remote port')
    if any(int(i['publicPort']) == public or int(i['remotePort']) == remote for i in data['instances']):
        raise SystemExit('error: requested port already assigned')
    if port_listening(public, '0.0.0.0') or port_listening(remote, '127.0.0.1'):
        raise SystemExit('error: requested port already in use')
    password = password_from_args(args)
    user_dir = USERS / user
    user_dir.mkdir(parents=True, exist_ok=False)
    ensure_permissions(user_dir)
    (user_dir / 'htpasswd').write_text(f'{user}:{htpasswd_hash(password)}\n')
    set_nginx_readable(user_dir / 'htpasswd')
    os.chmod(user_dir / 'htpasswd', 0o640)
    (user_dir / 'frpc.toml').write_text(frpc_conf(user, remote))
    os.chmod(user_dir / 'frpc.toml', 0o600)
    conf_path = NGINX_DIR / f'gardener-user-{user}.conf'
    conf_path.write_text(nginx_conf(user, public, remote))
    instance = {
        'user': user,
        'publicPort': public,
        'remotePort': remote,
        'proxyName': f'gardener-{relay_id_for_user(user)}',
        'url': f'http://{SERVER_ADDR}:{public}',
        'basicAuthUser': user,
        'createdAt': time.strftime('%Y-%m-%dT%H:%M:%S%z'),
        'nginxConf': str(conf_path),
        'frpcConfig': str(user_dir / 'frpc.toml'),
    }
    try:
        setup_key, provision_path = write_provision(instance, password, setup_key_from_args(args))
        instance['setupKey'] = setup_key
        instance['provisionUrl'] = f'{RELAY_PUBLIC_BASE_URL}/downloads/provision/{setup_key}/gardener.provision.json'
        instance['provisionPath'] = str(provision_path)
        data['instances'].append(instance)
        data['instances'].sort(key=lambda x: int(x['publicPort']))
        save_state(data)
        reload_nginx()
    except Exception:
        data['instances'] = [i for i in data.get('instances', []) if i.get('user') != user]
        save_state(data)
        shutil.rmtree(user_dir, ignore_errors=True)
        if 'setup_key' in locals():
            shutil.rmtree(PROVISION_ROOT / setup_key, ignore_errors=True)
        conf_path.unlink(missing_ok=True)
        raise
    result = {**instance, 'password': password, 'installCommand': install_command(setup_key), 'macInstallCommand': mac_install_command(setup_key)}
    if not args.show_secrets:
        result = redact_add_result(result)
    print(json.dumps(result, ensure_ascii=False, indent=2))


def remove_user(args):
    user = sanitize_user(args.user)
    data = load_state()
    found = [i for i in data['instances'] if i['user'] == user]
    if not found:
        raise SystemExit(f'error: user {user} not found')
    inst = found[0]
    data['instances'] = [i for i in data['instances'] if i['user'] != user]
    save_state(data)
    (NGINX_DIR / f'gardener-user-{user}.conf').unlink(missing_ok=True)
    shutil.rmtree(USERS / user, ignore_errors=True)
    if inst.get('setupKey'):
        shutil.rmtree(PROVISION_ROOT / inst['setupKey'], ignore_errors=True)
    reload_nginx()
    print(f'removed {user}')


def redacted_list_state(data):
    out = {'instances': []}
    for item in data.get('instances', []):
        redacted = dict(item)
        if redacted.get('setupKey'):
            redacted['setupKey'] = '<redacted>'
        if redacted.get('provisionPath'):
            redacted['provisionPath'] = '<redacted>'
        out['instances'].append(redacted)
    return out


def list_users(args):
    data = load_state()
    rows = data['instances']
    if args.json:
        print(json.dumps(redacted_list_state(data), ensure_ascii=False, indent=2))
        return
    if not rows:
        print('no users')
        return
    print(f'USER{"":14} URL{"":27} REMOTE  PROXY              STATUS    SETUP')
    for i in rows:
        online = 'online' if port_listening(int(i['remotePort']), '127.0.0.1') else 'offline'
        setup = 'present' if i.get('setupKey') else '-'
        print(f'{i["user"]:<18} {i["url"]:<32} {i["remotePort"]:<7} {i["proxyName"]:<18} {online:<9} {setup}')


def show_user(args):
    user = sanitize_user(args.user)
    data = load_state()
    for i in data['instances']:
        if i['user'] == user:
            out = dict(i)
            if args.with_frpc:
                out['frpc'] = (USERS / user / 'frpc.toml').read_text()
            if args.with_provision:
                if not args.show_secrets:
                    raise SystemExit('error: --with-provision prints relay passwords and frp tokens; add --show-secrets to confirm')
                if not out.get('provisionPath') or not Path(out['provisionPath']).exists():
                    raise SystemExit('error: this user has no provision file; rotate/reset the user to create one')
                out['provision'] = json.loads(Path(out['provisionPath']).read_text())
            if out.get('setupKey') and args.with_setup_key:
                require_relay_configured()
                out['installCommand'] = install_command(out['setupKey'])
                out['macInstallCommand'] = mac_install_command(out['setupKey'])
            elif out.get('setupKey'):
                out['setupKey'] = '<redacted>'
                if out.get('provisionPath'):
                    out['provisionPath'] = '<redacted>'
            print(json.dumps(out, ensure_ascii=False, indent=2))
            return
    raise SystemExit(f'error: user {user} not found')


def main():
    p = argparse.ArgumentParser(prog='gardener-relay')
    sub = p.add_subparsers(dest='cmd', required=True)
    a = sub.add_parser('add')
    a.add_argument('user')
    a.add_argument('--public-port', type=int)
    a.add_argument('--remote-port', type=int)
    a.add_argument('--password')
    a.add_argument('--password-file', help='read web login password from a local file instead of the command line')
    a.add_argument('--setup-key', help='optional preselected secret setup key, default: random sk_*')
    a.add_argument('--setup-key-file', help='read optional preselected setup key from a local file')
    a.add_argument('--show-secrets', action='store_true', help='print generated password, setup key and install commands')
    a.set_defaults(func=add_user)
    r = sub.add_parser('remove')
    r.add_argument('user')
    r.set_defaults(func=remove_user)
    l = sub.add_parser('list')
    l.add_argument('--json', action='store_true')
    l.set_defaults(func=list_users)
    g = sub.add_parser('gc-provisions')
    g.set_defaults(func=gc_provisions)
    s = sub.add_parser('show')
    s.add_argument('user')
    s.add_argument('--with-frpc', action='store_true')
    s.add_argument('--with-provision', action='store_true')
    s.add_argument('--with-setup-key', action='store_true', help='include setup key and install commands in output')
    s.add_argument('--show-secrets', action='store_true', help='confirm printing provision secrets')
    s.set_defaults(func=show_user)
    args = p.parse_args()
    args.func(args)

if __name__ == '__main__':
    main()
