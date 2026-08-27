# Unit tests for Test::Nginx::Socket::gen_tls_client_cmd_from_req
use lib 'lib';
use Test::Nginx::Socket tests => 7;

local $Test::Nginx::Util::TlsClient = '/opt/my-client';
local $Test::Nginx::Util::TlsClientMode = 1;
local $Test::Nginx::Util::TlsClientOptions = undef;
local $Test::Nginx::Util::TlsClientVerify = undef;
local $Test::Nginx::Util::ServerAddr = '127.0.0.1';
local $Test::Nginx::Util::ServerPortForClient = 1984;
local $Test::Nginx::Util::ServerName = 'localhost';
local $Test::Nginx::Util::Verbose = undef;
local $Test::Nginx::Util::UseHttp2 = undef;

sub block_named {
    my $name = shift;
    for my $b (blocks()) {
        return $b if $b->name eq $name;
    }
    die "no test block named $name";
}

sub cmd_for {
    my $name = shift;
    my $b = block_named($name);
    my $reqs = Test::Nginx::Socket::get_req_from_block($b);
    return Test::Nginx::Socket::gen_tls_client_cmd_from_req($b, $reqs->[0]);
}

sub flag_value {
    my ($cmd, $flag) = @_;
    for (my $i = 0; $i < $#$cmd; $i++) {
        return $cmd->[$i + 1] if $cmd->[$i] eq $flag;
    }
    return undef;
}

is_deeply(cmd_for('defaults'),
          ['/opt/my-client',
           '--addr', '127.0.0.1:1984',
           '--sni', 'localhost',
           '--timeout', timeout(),
           '--insecure'],
          'default argv is command plus addr/sni/timeout/insecure');

is_deeply(cmd_for('named plus options'),
          ['other-client',
           '--client', 'firefox',
           '--addr', '127.0.0.1:1984',
           '--sni', 'example.com',
           '--timeout', timeout(),
           '--insecure'],
          '--- tls_client and --- tls_client_options');

{
    my $cmd = cmd_for('timeout');
    is(flag_value($cmd, '--timeout'), 7, '--- timeout is passed in seconds');
}

{
    my $cmd = cmd_for('ipv6');
    is(flag_value($cmd, '--addr'), '[::1]:1984', 'IPv6 addresses are bracketed');
}

{
    my $cmd = cmd_for('verify');
    ok(!grep({ $_ eq '--insecure' } @$cmd), '--- tls_client_verify omits --insecure');
}

{
    local $Test::Nginx::Util::TlsClient = undef;
    is(Test::Nginx::Util::resolve_tls_client(), undef,
       'no default command name');
}

{
    local $Test::Nginx::Util::TlsClientOptions = '--profile chrome';
    my $cmd = cmd_for('defaults');
    ok(grep({ $_ eq '--profile' } @$cmd), 'TEST_NGINX_TLS_CLIENT_OPTIONS are inserted');
}

__DATA__

=== defaults
--- request
GET /t

=== named plus options
--- tls_client: other-client
--- tls_client_options: --client firefox
--- tls_client_sni: example.com
--- request
GET /t

=== timeout
--- timeout: 7
--- request
GET /t

=== ipv6
--- server_addr_for_client: ::1
--- request
GET /t

=== verify
--- tls_client_verify
--- request
GET /t
