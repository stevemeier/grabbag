#!/usr/bin/perl

use strict;
use warnings;
use Date::Parse;
use Getopt::Long;
use HTML::Table;
use XML::Simple;

$ENV{'LC_ALL'} = "C";
my @transactions;
my %tdetails;
my ($erratafile, $stylesheet, $limit);
my %pkg2errata;
my %errata;

my $getopt = GetOptions('errata=s'     => \$erratafile,
                        'stylesheet=s' => \$stylesheet,
                        'limit=i'      => \$limit);

# Load errata, if defined
if (defined($erratafile)) {
  print STDERR "Loading $erratafile\n";
  my $xml = XMLin($erratafile, ForceArray => [ qw(/keywords/ os_arch os_release packages) ] );
  foreach my $advisory (sort(keys(%{$xml}))) {
    foreach my $package (@{$xml->{$advisory}->{'packages'}}) {
      $pkg2errata{$package} = $advisory;
      $errata{$advisory}{'synopsis'} = $xml->{$advisory}->{'synopsis'};
      $errata{$advisory}{'issue_date'} = str2time($xml->{$advisory}->{'issue_date'});
    }
  }
}

# Find all yum transactions
open(HISTORYALL, '-|', '/usr/bin/yum history list all');
while(<HISTORYALL>) {
  if (/\s+(\d+)\s+\|/) {
    push(@transactions, $1);
  }
}
close(HISTORYALL);

print STDERR "Found ".scalar(@transactions)." YUM transactions\n";
if ($limit) {
  # Only process limited number of transactions
  @transactions = splice(@transactions, 0, $limit);
}

# Process each transaction
foreach my $transaction (@transactions) {
  my $pa = 0;
  my $updated;
  my $oldver;

  #print STDERR "Details for transaction $transaction:\n";
  open(HISTINFO, '-|', "/usr/bin/yum history info $transaction");
  while(<HISTINFO>) {
    if (/^Begin time\s+: (.*?)$/) { $tdetails{$transaction}{'start'} = $1 }
    if (/^End time\s+:\s+(.*?) \(/) { $tdetails{$transaction}{'end'} = $1 }
    if ((/^User/) && (/\<(.*?)\>$/)) { $tdetails{$transaction}{'username'} = $1 }
    if (/^Command Line\s+: (.*?)$/) { $tdetails{$transaction}{'command'} = $1 }
    if (/^Return-Code\s+: (.*?)$/) { 
      if ($1 eq 'Success') { 
        $tdetails{$transaction}{'rc'} = 0; 
      } else {
        $tdetails{$transaction}{'rc'} = 1;
      }
    }
    if (/^Packages Altered:/) { $pa = 1 }
    if ($pa) {
      # Obsoleting and Obsoleted don't add value here
      if (/^\s+Install\s+(.*?)$/) { push(@{$tdetails{$transaction}{'install'}}, &strip_repo($1)); }
      if (/^\s+Dep-Install\s+(.*?)$/) { push(@{$tdetails{$transaction}{'dep-install'}}, &strip_repo($1)) }
      if (/^\s+Erase\s+(.*?)$/) { push(@{$tdetails{$transaction}{'erase'}}, &strip_repo($1)) }
      if (/^\s+Reinstall\s+(.*?)$/) { push(@{$tdetails{$transaction}{'reinstall'}}, &strip_repo($1)) }
      if (/^\s+Updated\s+(.*?)$/) { $oldver = $1 }
      if (($oldver) && (/^\s+Update\s+(.*?)$/)) { 
        push(@{$tdetails{$transaction}{'update'}}, ([&strip_repo($oldver), &strip_repo($1)]));
        undef($oldver);
      }

      if (/^Downgrade\s+(.*?)$/) { $oldver = $1 }
      if (($oldver) && (/^\s+Downgraded\s+(.*?)$/)) {
        push(@{$tdetails{$transaction}{'downgrade'}}, ([&strip_repo($1), &strip_repo($oldver)]));
        undef($oldver);
      }
    }

  }
  close(HISTINFO);
  # Fix end date
  if ($tdetails{$transaction}{'end'} eq '') {
    $tdetails{$transaction}{'end'} = $tdetails{$transaction}{'start'}
  }
  if (length($tdetails{$transaction}{'end'}) == 13) {
    $tdetails{$transaction}{'end'} = substr($tdetails{$transaction}{'start'}, 0, (length($tdetails{$transaction}{'start'}) - 13)) . $tdetails{$transaction}{'end'};
  }

  # Find related errata
  push(@{$tdetails{$transaction}{'errata'}}, &find_errata(\%{$tdetails{$transaction}}));
}

print "<html><head>\n";
if ($stylesheet) {
  print "<link rel=\"stylesheet\" href=\"$stylesheet\">\n";
} else {
  print '<style type="text/css" media="all">'."\n";
  print "td { font-family: Verdana, Geneva, sans-serif; }\n";
  print "td.header { background-color: #222222; color: #ffffff; font-size: 200%; }\n";
  print "td.action { background-color: #888888; color: #ffffff; font-size: 150%; }\n";
  print "td.errata { background-color: #228822; color: #ffffff; font-size: 150%; }\n";
  print ".blank_row { height: 50px; }\n";
  print ".styled-table { border-collapse: collapse; margin: 25px 0; min-width: 400px; box-shadow: 0 0 20px rgba(0, 0, 0, 0.15); }\n";
  print ".styled-table td { padding: 2px 15px; }\n";
  print ".styled-table tbody tr { border-bottom: 1px solid #dddddd; }\n";
  print "</style>\n";
}
print "</head><body>\n";

#my $table = new HTML::Table(-width => '80%');
foreach my $transaction (@transactions) {
  my $table = new HTML::Table(-width => '80%', -class => 'styled-table');
  $table->addRow("Transaction #$transaction");
  $table->setCellAttr(-1, 1, 'class="header"');
  $table->setCellColSpan(-1, 1, 2);
  $table->addRow("Started at:", $tdetails{$transaction}{'start'});
  $table->addRow("Finished at:", $tdetails{$transaction}{'end'});
  if (length($tdetails{$transaction}{'command'}) <= 40) {
    $table->addRow("Parameters:", $tdetails{$transaction}{'command'});
  } else {
    $table->addRow("Parameters:");
    $table->setCellColSpan(-1, 1, 2);
    $table->addRow($tdetails{$transaction}{'command'});
    $table->setCellColSpan(-1, 1, 2);
  }
  $table->addRow("Result:", $tdetails{$transaction}{'rc'});
  $table->addRow("Username:", $tdetails{$transaction}{'username'});

  if (defined($tdetails{$transaction}{'errata'})) {
    if (scalar(@{$tdetails{$transaction}{'errata'}}) > 0) {
      $table->addRow('Related errata');
      $table->setCellAttr(-1, 1, 'class="errata"');
      $table->setCellColSpan(-1, 1, 2);
      foreach $_ (sort(@{$tdetails{$transaction}{'errata'}})) {
        my $exposure = undef;
        if (/CESA/) {
          # Calculate exposure time
          # print STDERR "Calculating exposure for $_\n";
          $exposure = str2time($tdetails{$transaction}{'end'}) - $errata{$_}{'issue_date'};
          if ($exposure < 86400) {
            $exposure = "less than 1 day after release";
          } elsif (($exposure >= 86400) && ($exposure < 172800)) {
            $exposure = "1 day after release";
          } else {
            $exposure = (int($exposure / 86400)) . " days after release";
          }
        }

        (my $pretty = $_) =~ s/--/:/;
        if ($exposure) {
          $table->addRow($pretty." [$exposure]", $errata{$_}{'synopsis'});
        } else {
          $table->addRow($pretty, $errata{$_}{'synopsis'});
        }
      }
    }
  }

  if (defined($tdetails{$transaction}{'erase'})) {
    $table->addRow('Packages removed');
    $table->setCellAttr(-1, 1, 'class="action"');
    $table->setCellColSpan(-1, 1, 2);
    foreach $_ (@{$tdetails{$transaction}{'erase'}}) {
      $table->addRow($_);
    }
  }

  if (defined($tdetails{$transaction}{'downgrade'})) {
    $table->addRow('Packages downgraded');
    $table->setCellAttr(-1, 1, 'class="action"');
    $table->setCellColSpan(-1, 1, 2);
    foreach $_ (@{$tdetails{$transaction}{'downgrade'}}) {
      $table->addRow(@{$_});
    }
  }

  if (defined($tdetails{$transaction}{'dep-install'})) {
    $table->addRow('Packages installed for dependencies');
    $table->setCellAttr(-1, 1, 'class="action"');
    $table->setCellColSpan(-1, 1, 2);
    foreach $_ (@{$tdetails{$transaction}{'dep-install'}}) {
      $table->addRow($_);
    }
  }

  if (defined($tdetails{$transaction}{'install'})) {
    $table->addRow('Packages installed');
    $table->setCellAttr(-1, 1, 'class="action"');
    $table->setCellColSpan(-1, 1, 2);
    foreach $_ (@{$tdetails{$transaction}{'install'}}) {
      $table->addRow($_);
    }
  }

  if (defined($tdetails{$transaction}{'update'})) {
    $table->addRow('Packages updated');
    $table->setCellAttr(-1, 1, 'class="action"');
    $table->setCellColSpan(-1, 1, 2);
    foreach $_ (@{$tdetails{$transaction}{'update'}}) {
      $table->addRow(@{$_});
      $table->setCellWidth(-1, 1, "50%");
      $table->setCellWidth(-1, 2, "50%");
    }
  }

  # Add spacing to next transaction
# $table->addRow();
# $table->setCellAttr(-1, 1, 'class="blank_row"');
# $table->setCellColSpan(-1, 1, 2);

  $table->print;
}

#$table->print;
print "</body></html>\n";
exit;

sub strip_repo {
  my $input = shift;
  $input =~ s/\s+\@.*$//;
  chomp($input);

  return $input;
}

sub uniq {
  my (@input) = @_;
  my %all = ();
  @all{@input} = 1;
  return (keys %all);
}

sub find_errata {
  my %transaction = %{$_[0]};
  my @result;

  foreach $_ (@{$transaction{'update'}}) {
    # Parse the previous version to get the name
    my %info = &parse_nevra(@{$_}[0]);
    my $rpm = scalar($info{'name'});
    # Remove `epoch`, as it's not part of the RPM filename
       $rpm .= "-".strip_epoch(@{$_}[1]).".rpm";
    if (defined($pkg2errata{$rpm})) {
      push(@result, $pkg2errata{$rpm});
    # print STDERR "$rpm matched $pkg2errata{$rpm}\n";
    }
  }

  return uniq(@result);
}

sub strip_epoch {
  my $input = shift;
  $input =~ s/^.*?\://;

  return $input;
}

sub parse_nevra {
  my ( $str ) = shift;

  my $arch = ( split( /\./, $str ) )[-1];
  $str =~ s/\.$arch$//;

  my $rel = ( split( /-/, $str ) )[-1];
  $str =~ s/-$rel$//;

  my $ver_str = ( split( /-/, $str ) )[-1];
  my ( $epoch, $ver ) = split( /:/, $ver_str );
  my $trimmer;

  if ( !defined($ver) ) {    # no epoch
      $ver     = $epoch;
      $epoch   = undef;
      $trimmer = $ver;
  }
  else {
      $trimmer = "$epoch:$ver";
  }
  $str =~ s/-\Q$trimmer\E//;

  my %info;
  @info{qw(name arch rel ver epoch)} = ( $str, $arch, $rel, $ver, $epoch );
  return %info;
}

