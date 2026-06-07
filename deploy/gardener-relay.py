#!/usr/bin/env python3
import argparse, base64, crypt, json, os, re, secrets, shutil, socket, string, subprocess, sys, time
from pathlib import Path

ROOT = Path('/etc/gardener-relay')
USERS = ROOT / 'users'
STATE = ROOT / 'instances.json'
NGINX_DIR = Path('/etc/nginx/conf.d')
FRPS_CONF = Path('/etc/frp/frps.toml')
DOWNLOAD_ROOT = Path('/srv/gardener-downloads/public')
PROVISION_ROOT = DOWNLOAD_ROOT / 'provision'
MAX_RELAY_DOWNLOAD_URL_LENGTH = 2048
SERVER_ADDR = os.environ.get('GARDENER_RELAY_SERVER_ADDR', 'YOUR_RELAY_SERVER')
FRPS_PORT = int(os.environ.get('GARDENER_RELAY_FRPS_PORT', '27000'))
PUBLIC_START = int(os.environ.get('GARDENER_RELAY_PUBLIC_START', '28081'))
PUBLIC_END = int(os.environ.get('GARDENER_RELAY_PUBLIC_END', '28100'))
REMOTE_START = int(os.environ.get('GARDENER_RELAY_REMOTE_START', '18081'))
REMOTE_END = int(os.environ.get('GARDENER_RELAY_REMOTE_END', '18100'))
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
    validate_download_url_lengths()


def validate_download_url_lengths():
    urls = {
        'windows package URL': PACKAGE_URL,
        'windows install script URL': INSTALL_SCRIPT_URL,
        'macOS install script URL': MAC_INSTALL_SCRIPT_URL,
    }
    for arch, url in MAC_PACKAGE_URLS.items():
        urls[f'macOS {arch} package URL'] = url
    for label, url in urls.items():
        if len(url) > MAX_RELAY_DOWNLOAD_URL_LENGTH:
            raise SystemExit(f'error: {label} is too long')

def run(cmd, check=True):
    return subprocess.run(cmd, check=check, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)


def load_state():
    if not STATE.exists():
        return {'instances': []}
    return json.loads(STATE.read_text())


def save_state(data):
    ROOT.mkdir(parents=True, exist_ok=True)
    tmp = STATE.with_suffix('.tmp')
    tmp.write_text(json.dumps(data, ensure_ascii=False, indent=2) + '\n')
    tmp.replace(STATE)


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


def htpasswd_hash(password):
    salt = '$6$' + secrets.token_urlsafe(12)
    return crypt.crypt(password, salt)


def nginx_conf(user, public_port, remote_port):
    return f'''server {{
    listen {public_port};
    server_name _;

    client_max_body_size 200m;

    auth_basic "Gardener {user}";
    auth_basic_user_file /etc/gardener-relay/users/{user}/htpasswd;

    location / {{
        proxy_pass http://127.0.0.1:{remote_port};
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_buffering off;
        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;
    }}
}}
'''


def frpc_conf(user, remote_port):
    token = read_frp_token()
    return f'''serverAddr = "{SERVER_ADDR}"
serverPort = {FRPS_PORT}

auth.method = "token"
auth.token = "{token}"

[[proxies]]
name = "gardener-{relay_id_for_user(user)}"
type = "tcp"
localIP = "127.0.0.1"
localPort = 8080
remotePort = {remote_port}
'''


def set_nginx_readable(path):
    try:
        import grp
        gid = grp.getgrnam('nginx').gr_gid
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


def write_provision(instance, password, setup_key=None):
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
    os.chmod(provision_dir, 0o755)
    os.chmod(path, 0o644)
    return setup_key, path


def install_command(setup_key):
    return ('powershell -ExecutionPolicy Bypass -Command '
            f'"iwr {INSTALL_SCRIPT_URL} -OutFile install-gardener.ps1; '
            f'.\\install-gardener.ps1 -RelayBaseUrl {RELAY_PUBLIC_BASE_URL} -SetupKey {setup_key} -DesktopShortcut -StartMenuShortcut -StartAfterInstall"')


def mac_install_command(setup_key):
    return f'curl -fsSL {MAC_INSTALL_SCRIPT_URL} -o install-gardener-macos.sh && bash install-gardener-macos.sh --relay-base-url {RELAY_PUBLIC_BASE_URL} --setup-key {setup_key}'


def reload_nginx():
    test = run(['nginx', '-t'], check=False)
    if test.returncode != 0:
        raise SystemExit(test.stderr + test.stdout)
    run(['systemctl', 'reload', 'nginx'])


def add_user(args):
    require_relay_configured()
    user = sanitize_user(args.user)
    data = load_state()
    if any(i['user'] == user for i in data['instances']):
        raise SystemExit(f'error: user {user} already exists')
    proxy_name = f'gardener-{relay_id_for_user(user)}'
    if any(i.get('proxyName') == proxy_name for i in data['instances']):
        raise SystemExit(f'error: proxy name {proxy_name} already exists; choose another user name')
    public, remote = (args.public_port, args.remote_port) if args.public_port and args.remote_port else find_free_slot(data['instances'])
    if any(int(i['publicPort']) == public or int(i['remotePort']) == remote for i in data['instances']):
        raise SystemExit('error: requested port already assigned')
    if port_listening(public, '0.0.0.0') or port_listening(remote, '127.0.0.1'):
        raise SystemExit('error: requested port already in use')
    password = args.password or make_password()
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
        setup_key, provision_path = write_provision(instance, password, args.setup_key)
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


def list_users(args):
    data = load_state()
    rows = data['instances']
    if args.json:
        print(json.dumps(data, ensure_ascii=False, indent=2))
        return
    if not rows:
        print('no users')
        return
    print(f'USER{"":14} URL{"":27} REMOTE  PROXY              STATUS    SETUP')
    for i in rows:
        online = 'online' if port_listening(int(i['remotePort']), '127.0.0.1') else 'offline'
        setup = i.get('setupKey', '-')
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
                if not out.get('provisionPath') or not Path(out['provisionPath']).exists():
                    raise SystemExit('error: this user has no provision file; rotate/reset the user to create one')
                out['provision'] = json.loads(Path(out['provisionPath']).read_text())
            if out.get('setupKey'):
                require_relay_configured()
                out['installCommand'] = install_command(out['setupKey'])
                out['macInstallCommand'] = mac_install_command(out['setupKey'])
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
    a.add_argument('--setup-key', help='optional preselected secret setup key, default: random sk_*')
    a.set_defaults(func=add_user)
    r = sub.add_parser('remove')
    r.add_argument('user')
    r.set_defaults(func=remove_user)
    l = sub.add_parser('list')
    l.add_argument('--json', action='store_true')
    l.set_defaults(func=list_users)
    s = sub.add_parser('show')
    s.add_argument('user')
    s.add_argument('--with-frpc', action='store_true')
    s.add_argument('--with-provision', action='store_true')
    s.set_defaults(func=show_user)
    args = p.parse_args()
    args.func(args)

if __name__ == '__main__':
    main()
