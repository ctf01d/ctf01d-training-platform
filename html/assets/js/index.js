
var entityMap = {
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#39;',
    '/': '&#x2F;',
    '`': '&#x60;',
    '=': '&#x3D;'
};

function escapeHtml(string) {
    return String(string).replace(/[&<>"'`=\/]/g, function (s) {
      return entityMap[s];
    });
}

function isRootPage(pathname) {
    return pathname == "/";
}

function isGamesPage(pathname) {
    return pathname == "/games" || pathname == "/games/";
}

function isServicesPage(pathname) {
    return pathname == "/services" || pathname == "/services/";
}

function isTeamsPage(pathname) {
    return pathname == "/teams" || pathname == "/teams/";
}

function isUsersPage(pathname) {
    return pathname == "/users" || pathname == "/users/";
}

function getHumanTimeHasPassed(dt_end) {
    var dt = dt_end.getTime();
    var dt_now = new Date().getTime();
    var diff = dt_now - dt;
    diff = Math.floor(diff / 1000); // sec
    if (diff < 60) {
        return diff + " seconds ago";
    }
    diff = Math.floor(diff / 60);
    if (diff < 60) {
        return diff + " minutes ago";
    }
    diff = Math.floor(diff / 60);
    if (diff < 24) {
        return diff + " hours ago";
    }
    diff = Math.floor(diff / 24);
    return diff + " days ago";
}

function showSuccessNotification(text) {
    $.notify({
        icon: "add_alert",
        message: text
    }, {
        type: 'success',
        timer: 4000,
        placement: {
            from: 'top',
            align: 'right'
        }
    });
  }

function updateGameTeams(el_id, game_id) {
    window.ctf01d_tp_api.game_info(game_id).fail(function(res) {
        $('#games_page_error').css({
            "display": "block"
        });
        $('#games_page_error').html("Error load info about game #" + game_id);
        console.error(res);
    }).done(function(res) {
        console.log(res)
        var gameTeamsInfoHtml = '<ul class="list-group">';


        for (var i in res['team_details']) {
            var team_info = res['team_details'][i];
            console.log("team_info", team_info);
            gameTeamsInfoHtml += '<li class="list-group-item"><strong>' + escapeHtml(team_info.name) + '</strong> - ' + escapeHtml(team_info.description) + '</li>';
            console.log(team_info)
        }
        gameTeamsInfoHtml += '</ul>';
        $('#' + el_id).html(gameTeamsInfoHtml);
    })
}

function gameCreate() {
    var startTimeInput = $('#game_create_start_time').val();
    var endTimeInput = $('#game_create_end_time').val();
    // convert to ISO 8601
    var startTime = new Date(startTimeInput).toISOString();
    var endTime = new Date(endTimeInput).toISOString();
    var description = $('#game_create_description').val()
    window.ctf01d_tp_api.game_create({
        start_time: startTime,
        end_time: endTime,
        description: description,
    }).fail(function(res) {
        $('#games_page_error').css({
            "display": "block"
        });
        $('#games_page_error').html("Error creating game");
        console.error(res);
    }).done(function(res) {
        console.log(res)
        $('#modal_create_game').modal('hide');
        showSuccessNotification('Game created successfully!')
        renderGamePage();
    })
}

function showLoginForm() {
    $('#sign_error_info').html('')
    $('#sign_error_info').css({"display": "none"});
    $('#sign_in_username').focus();
    $('#sign_in_username').unbind();
    $('#sign_in_username').keypress(function (e) {
        if (e.which == 13) {
            $('#sign_in_password').focus();
            return false; // <---- Add this line
        }
    });
    $('#sign_in_password').unbind();
    $('#sign_in_password').keypress(function (e) {
        if (e.which == 13) {
            doSignIn();
            return false; // <---- Add this line
        }
    });
    $('#modal_sign_in').modal('show');
}

function doSignIn() {
    var username = $('#sign_in_username').val();
    var password = $('#sign_in_password').val();
    $('#sign_error_info').html('')
    $('#sign_error_info').css({"display": "none"});
    window.ctf01d_tp_api.auth_signin({
        user_name: username,
        password: password,
    }).fail(function(res) {
        $('#sign_error_info').css({"display": ""});
        $('#sign_error_info').html('SignIn failed. Check username and password.');
        console.error(res);
    }).done(function(res) {
        console.log(res);
        showSuccessNotification('Login successful!');
        $('#modal_sign_in').modal('hide');
        setTimeout(function () {
            window.location.reload();
        }, 1000);
    })
}

function doSignOut() {
    window.ctf01d_tp_api.auth_signout().fail(function(res) {
        console.error(res);
    }).done(function(res) {
        showSuccessNotification('Sing out successfull!');
        window.location.reload();
    })
}

$(document).ready(function () {
    $(document).on('click', '.btn-info', function () {
        var gameId = $(this).closest('.list-group-item').find('h5').text().substring(1);
        fetchGameResults(gameId);
    });
});

function fetchGameResults(gameId) {
    // просто для проверки гипотезы, потом убрать в -> window.ctf01d_tp_api
    $.ajax({
        url: '/api/v1/games/' + gameId + '/scoreboard',
        method: 'GET',
        success: function (data) {
            renderGameResults(data);
            $('#modal_game_results').modal('show');
        },
        error: function (xhr, status, error) {
            console.error('Failed to fetch game results:', error);
        }
    });
}

function escapeHtml(text) {
    if (typeof text !== 'string') {
        return text;
    }

    var map = {
        '&': '&amp;',
        '<': '&lt;',
        '>': '&gt;',
        '"': '&quot;',
        "'": '&#039;'
    };
    return text.replace(/[&<>"']/g, function (m) { return map[m]; });
}

function renderGameResults(results) {
    var resultsHtml = '<table class="table table-striped">';
    resultsHtml += '<thead><tr><th>Rank</th><th>Score</th><th>Team Id</th></tr></thead><tbody>';

    results.forEach(function (result) {
        resultsHtml += '<tr>';
        resultsHtml += '<td>' + escapeHtml(result.rank) + '</td>';
        resultsHtml += '<td>' + result.score.toFixed(2) + '</td>';
        resultsHtml += '<td>' + escapeHtml(result.team_id) + '</td>';
        resultsHtml += '</tr>';
    });

    resultsHtml += '</tbody></table>';
    $('#game_results_content').html(resultsHtml);
}

function renderGamesPage() {
    $('.spa-web-page').css({"display": ""})
    $('#games_page').css({"display": "block"})
    if (window.location.pathname != "/games/") {
        window.location.href = "/games/";
    }
    window.ctf01d_tp_api.games_list().fail(function(res) {
        $('#games_page_error').css({
            "display": "block"
        });
        $('#games_page_error').html("Error loading games");
        console.error(res);
    }).done(function(res) {
        var gamesHtml = ""
        for (var i in res) {
            var game_info = res[i];
            // console.log("game_info", game_info);
            gamesHtml += '<div href="#" class="list-group-item list-group-item-action flex-column align-items-start">';
            gamesHtml += '  <div class="d-flex w-100 justify-content-between">';
            gamesHtml += '    <h5 class="mb-1">#' + game_info.id + '</h5>';
            gamesHtml += '    <button type="button" class="btn btn-info">' + tr('Результаты игры') + '</button>';
            gamesHtml += '    <small>' + getHumanTimeHasPassed(new Date(game_info.end_time)) + '</small>';
            gamesHtml += '  </div>';
            gamesHtml += '  <p class="mb-1">' + escapeHtml(game_info.description) + '</p>';
            gamesHtml += '  <small>Начало: ' + new Date(game_info.start_time) + '</small><br>';
            gamesHtml += '  <small>Конец: ' + new Date(game_info.end_time) + '</small><br><br>';
            gamesHtml += '  <div id="game_teams_' + game_info.id + '"> ' + new Date(game_info.end_time) + '</div>';
            gamesHtml += '</div>';
            updateGameTeams('game_teams_' + game_info.id, game_info.id)
        }
        $('#games_page_list').html(gamesHtml);
    })
}

function serviceCreateOrUpdate() {
    $('#services_page_error').css({"display": "none"});
    $('#services_page_error').html("");

    var service_id = $('#service_update_service_id').val();

    var serviceName = $('#service_create_name').val();
    var serviceAuthor = $('#service_create_author').val();
    var serviceLogoUrl = $('#service_create_logo_url').val();
    var serviceDescription = $('#service_create_description').val();
    var serviceIsPublic = $('#service_create_is_public').prop('checked')
    if (service_id == 0) {
        window.ctf01d_tp_api.service_create({
            name: serviceName,
            author: serviceAuthor,
            logo_url: serviceLogoUrl,
            description: serviceDescription,
            is_public: serviceIsPublic,
        }).fail(function(res) {
            $('#services_page_error').css({
                "display": "block"
            });
            $('#services_page_error').html("Error creating service");
            console.error(res);
        }).done(function(res) {
            console.log(res)
            $('#modal_edit_or_create_service').modal('hide');
            showSuccessNotification('Service created successfully!')
            renderServicesPage();
        })
    } else {
        window.ctf01d_tp_api.service_update(service_id, {
            name: serviceName,
            author: serviceAuthor,
            logo_url: serviceLogoUrl,
            description: serviceDescription,
            is_public: serviceIsPublic,
        }).fail(function(res) {
            $('#services_page_error').css({
                "display": "block"
            });
            $('#services_page_error').html("Error updating service");
            console.error(res);
        }).done(function(res) {
            console.log(res)
            $('#modal_edit_or_create_service').modal('hide');
            showSuccessNotification('Service updated successfully!')
            renderServicesPage();
        })
    }
}

function deleteService(service_id) {
    $('#services_page_error').css({"display": "none"});
    $('#services_page_error').html("");
    window.ctf01d_tp_api.service_delete(service_id).fail(function(res) {
        $('#services_page_error').css({
            "display": "block"
        });
        $('#services_page_error').html("Error delete service");
        console.error("services_list", res);
    }).done(function(res) {
        showSuccessNotification('Service deleted successfully!')
        renderServicesPage();
    })
}

function showCreateService() {
    $('#services_page_error').css({"display": "none"});
    $('#services_page_error').html("");

    $('#service_update_service_id').val(0);
    $('#service_create_name').val("");
    $('#service_create_author').val("");
    $('#service_create_logo_url').val("");
    $('#service_create_description').val("");
    $('#service_create_is_public').prop('checked', false);
    $('#btn_service_create_or_update').html("Create");
    $('#title_service_create_or_update').html("New Service");
    $('#modal_edit_or_create_service').modal('show');
}

function showUpdateService(service_id) {
    $('#services_page_error').css({"display": "none"});
    $('#services_page_error').html("");
    $('#btn_service_create_or_update').html("Update");

    window.ctf01d_tp_api.service_info(service_id).fail(function(res) {
        $('#services_page_error').css({
            "display": "block"
        });
        $('#services_page_error').html("Error get service info");
    }).done(function(res) {
        console.log("showUpdateService", res)
        $('#service_update_service_id').val(res["id"])
        $('#title_service_create_or_update').html("Update Service #" + res["id"]);
        $('#service_create_name').val(res["name"]);
        $('#service_create_author').val(res["name"]);
        $('#service_create_logo_url').val(res["logo_url"]);
        $('#service_create_description').val(res["description"]);
        if (res["is_public"]) {
            $('#service_create_is_public').prop('checked', true);
        } else {
            $('#service_create_is_public').prop('checked', false);
        }
        $('#modal_edit_or_create_service').modal('show');
    })
}

function renderServicesPage() {
    $('.spa-web-page').css({"display": ""})
    $('#services_page').css({"display": "block"})
    $('#services_page_error').css({"display": "none"});
    $('#services_page_error').html("");
    if (window.location.pathname != "/services/") {
        window.location.href = "/services/";
    }
    window.ctf01d_tp_api.services_list().fail(function(res) {
        $('#services_page_error').css({
            "display": "block"
        });
        $('#services_page_error').html("Error loading services");
        console.error("services_list", res);
    }).done(function(res) {
        var servicesHtml = ""
        for (var i in res) {
            var service_info = res[i];
            console.log("service_info", service_info);
            servicesHtml += '<div class="card services-card" style="width: 18rem;">';
            servicesHtml += '  <img class="card-img-top" src="' + service_info.logo_url + '" alt="Image of service">';
            servicesHtml += '  <div class="card-body">';
            servicesHtml += '    <h5 class="card-title">#' + service_info.id + ' ' + escapeHtml(service_info.name) + '</h5>'; // TODO uuid
            servicesHtml += '    <p class="card-text">' + escapeHtml(service_info.description) + '</p>';
            servicesHtml += '    <p class="card-text"> by ' + escapeHtml(service_info.author) + '</p>';
            // TODO
            // servicesHtml += '    <small>' + getHumanTimeHasPassed(new Date(game_info.end_time)) + '</small>';
            servicesHtml += '  </div>';
            // servicesHtml += '  <ul class="list-group list-group-flush">';
            // servicesHtml += '    <li class="list-group-item">Cras justo odio</li>';
            // servicesHtml += '    <li class="list-group-item">Dapibus ac facilisis in</li>';
            // servicesHtml += '    <li class="list-group-item">Vestibulum at eros</li>';
            // servicesHtml += '  </ul>';
            servicesHtml += '  <div class="card-body">';
            servicesHtml += '    <button class="btn btn-primary" onclick="showUpdateService(\'' + service_info.id + '\');">Update</button>';
            servicesHtml += '    <button class="btn btn-danger" onclick="deleteService(\'' + service_info.id + '\');">Delete</button>';
            servicesHtml += '  </div>';
            servicesHtml += '</div>';

            // servicesHtml += '  <div id="game_teams_' + game_info.id + '"> ' + new Date(game_info.end_time) + '</div>';
            // servicesHtml += '</div>';
            // updateGameTeams('service_teams_' + game_info.id, game_info.id)
        }
        $('#services_page_list').html(servicesHtml);
    })
}

function showCreateUser() {
    $('#users_page_error').css({ "display": "none" });
    $('#users_page_error').html("");

    $('#user_update_user_id').val(0);
    $('#user_create_name').val("");
    $('#user_create_display_name').val("");
    $('#user_create_avatar_url').val("");
    $('#user_create_password').val("");
    $('#user_create_status').val("");
    $('#user_create_role').val("guest");
    $('#user_create_team').empty(); // Clear teams

    loadTeams(); // Load teams into the select element

    $('#btn_user_create_or_update').html("Create");
    $('#title_user_create_or_update').html("New User");
    $('#modal_edit_or_create_user').modal('show');
}

function loadTeams() {
    window.ctf01d_tp_api.teams_list().done(function (teams) {
        var teamSelect = $('#user_create_team');
        teams.forEach(function (team) {
            teamSelect.append(`<option value="${team.id}">${team.name}</option>`);
        });
    }).fail(function () {
        alert('Error loading teams');
    });
}

function userCreateOrUpdate() {
    $('#users_page_error').css({ "display": "none" });
    $('#users_page_error').html("");

    var user_id = $('#user_update_user_id').val();

    var userName = $('#user_create_name').val();
    var userDisplayName = $('#user_create_display_name').val();
    var userAvatarUrl = $('#user_create_avatar_url').val();
    var userPassword = $('#user_create_password').val();
    var userStatus = $('#user_create_status').val();
    var userRole = $('#user_create_role').val();
    var userTeams = $('#user_create_team').val().map(function (id) {
        return id;
    });

    var userData = {
        user_name: userName,
        display_name: userDisplayName,
        avatar_url: userAvatarUrl,
        password: userPassword,
        status: userStatus,
        role: userRole,
        team_ids: userTeams
    };

    if (user_id == 0) {
        window.ctf01d_tp_api.user_create(userData).fail(function (res) {
            $('#users_page_error').css({ "display": "block" });
            $('#users_page_error').html("Error creating user");
            console.error(res);
        }).done(function (res) {
            $('#modal_edit_or_create_user').modal('hide');
            showSuccessNotification('User created successfully!');
            renderUsersPage();
        });
    } else {
        window.ctf01d_tp_api.user_update(user_id, userData).fail(function (res) {
            $('#users_page_error').css({ "display": "block" });
            $('#users_page_error').html("Error updating user");
            console.error(res);
        }).done(function (res) {
            $('#modal_edit_or_create_user').modal('hide');
            showSuccessNotification('User updated successfully!');
            renderUsersPage();
        });
    }
}

function renderUsersPage() {
    $('.spa-web-page').css({ "display": "" });
    $('#users_page').css({ "display": "block" });
    $('#users_page_error').css({ "display": "none" });
    $('#users_page_error').html("");
    if (window.location.pathname != "/users/") {
        window.location.href = "/users/";
    }
    window.ctf01d_tp_api.users_list().fail(function (res) {
        $('#users_page_error').css({ "display": "block" });
        $('#users_page_error').html("Error loading users");
        console.error("users_list", res);
    }).done(function (res) {
        var usersHtml = ""
        for (var i in res) {
            var user_info = res[i];
            usersHtml += '<div class="card services-card" style="width: 20rem;">';
            usersHtml += '  <div class="card-body">';
            usersHtml += '    <img class="users-card-avatar" src="' + user_info.avatar_url + '" alt="Image of user">';
            usersHtml += '    <h5 class="card-title">@' + user_info.user_name + ' - ' + escapeHtml(user_info.display_name) + '</h5>';
            usersHtml += '    <p class="card-text"> ' + escapeHtml(user_info.id) + '</p>';
            usersHtml += '    <p class="card-subtitle mb-2 text-muted"> role ' + escapeHtml(user_info.role) + '</p>';
            usersHtml += '    <p class="card-text"> state ' + escapeHtml(user_info.status) + '</p>';
            usersHtml += '  </div>';
            usersHtml += '  <div class="card-body">';
            usersHtml += '    <button class="btn btn-secondary" onclick="showMyTeams(\'' + user_info.id + '\');">Profile</button>';
            usersHtml += '    <button class="btn btn-primary" onclick="showUpdateUser(\'' + user_info.id + '\');">Update</button>';
            usersHtml += '    <button class="btn btn-danger" onclick="deleteUser(\'' + user_info.id + '\');">Delete</button>';
            usersHtml += '  </div>';
            usersHtml += '</div>';
        }
        $('#users_page_list').html(usersHtml);
    });
}

window.inited_autocomplete_universities = false;
window.cache_universities = {
    "": "00000000-0000-0000-0000-000000000000",
    "Без университета": "00000000-0000-0000-0000-000000000000",
}

function initAutocomleteUniversities() {
    if (window.inited_autocomplete_universities) {
        return;
    }
    window.inited_autocomplete_universities = true;

    // Docs look here: https://bootstrap-autocomplete.readthedocs.io/en/latest/

    $('#team_create_university').autoComplete({
        resolver: 'custom',
        minLength: 1,
        events: {
            search: function (qry, callback) {
                // let's do a custom ajax call
                ctf01d_tp_api.univercities_list(qry).fail(function() {
                    callback([
                        { "value": window.cache_universities[""], "text": "Без университета" },
                    ]);
                }).done(function(res) {
                    console.log(res);
                    var _list = [];
                    for (var i = 0; i < res.length; i++) {
                        var _guid = res[i]["id"];
                        var _name = res[i]["name"];
                        window.cache_universities[_name] = _guid;
                        _list.push({ "value": _guid, "text": _name });
                    }
                    _list.push({ "value": window.cache_universities[""], "text": "Без университета" });
                    callback(_list);
                });
            }
        }
    });
}

function showCreateTeam() {
    $('#teams_edit_or_create_team_error').css({ "display": "none" });
    $('#teams_edit_or_create_team_error').html("");

    initAutocomleteUniversities();

    $('#team_update_team_id').val("");
    $('#team_create_name').val("");
    $('#team_create_description').val("");
    $('#team_create_social_links').val("");
    $('#team_create_avatar_url').val("");
    $('#team_create_university').val("");
    $('#btn_team_create_or_update').html("Create");
    $('#title_team_create_or_update').html("New Team");
    $('#modal_edit_or_create_team').modal('show');
}

function showUpdateTeam(team_id) {
    $('#teams_edit_or_create_team_error').css({ "display": "none" });
    $('#teams_edit_or_create_team_error').html("");

    $('#title_team_create_or_update').html("Update Team");
    $('#btn_team_create_or_update').html("Update");

    console.log("team_id", team_id);
    initAutocomleteUniversities();

    $('#btn_service_create_or_update').html("Update");

    window.ctf01d_tp_api.team_info(team_id).fail(function(res) {
        $('#teams_edit_or_create_team_error').css({
            "display": "block"
        });
        $('#teams_edit_or_create_team_error').html("Error get team info");
    }).done(function(res) {
        console.log("showUpdateTeam", res)
        $('#team_update_team_id').val(res["id"]);
        $('#title_team_create_or_update').html("Update Team #" + res["id"]);

        $('#team_create_name').val(res["name"]);
        $('#team_create_description').val(res["description"]);
        $('#team_create_social_links').val(res["social_links"]);
        $('#team_create_avatar_url').val(res["avatar_url"]);
        $('#team_create_university').val(res["university"]);

        // fill cache
        ctf01d_tp_api.univercities_list(res["university"]).done(function(res) {
            for (var i = 0; i < res.length; i++) {
                var _guid = res[i]["id"];
                var _name = res[i]["name"];
                window.cache_universities[_name] = _guid;
                console.log(window.cache_universities)
            }
        })
        $('#modal_edit_or_create_team').modal('show');
    })
}

function deleteTeam(team_id) {
    $('#teams_page_error').css({"display": "none"});
    $('#teams_page_error').html("");
    window.ctf01d_tp_api.team_delete(team_id).fail(function(res) {
        $('#teams_page_error').css({
            "display": "block"
        });
        $('#teams_page_error').html("Error delete team");
        console.error("teams_list", res);
    }).done(function(res) {
        showSuccessNotification('Teams deleted successfully!')
        renderTeamsPage();
    })
}

function teamCreateOrUpdate() {
    $('#teams_edit_or_create_team_error').css({ "display": "none" });
    $('#teams_edit_or_create_team_error').html("");

    var team_id = $('#team_update_team_id').val();

    var team_name = $('#team_create_name').val();
    var team_description = $('#team_create_description').val();
    var team_social_links = $('#team_create_social_links').val();
    var team_avatar_url = $('#team_create_avatar_url').val();
    var team_university_name = $('#team_create_university').val();
    var team_university_id = window.cache_universities[team_university_name];

    var team_data = {
        name: team_name,
        description: team_description,
        social_links: team_social_links,
        avatar_url: team_avatar_url,
        university_id: team_university_id
    };

    if (team_id == "") {
        window.ctf01d_tp_api.team_create(team_data).fail(function (res) {
            $('#teams_edit_or_create_team_error').css({ "display": "block" });
            $('#teams_edit_or_create_team_error').html("Error creating team");
            console.error("team_create: ", res);
            if (res.responseJSON !== undefined) {
                $('#teams_edit_or_create_team_error').html("Error creating team: " + res.responseJSON["error"]);
            }
        }).done(function (res) {
            $('#modal_edit_or_create_team').modal('hide');
            showSuccessNotification('Team created successfully!');
            renderTeamsPage();
        });
    } else {
        window.ctf01d_tp_api.team_update(team_id, team_data).fail(function (res) {
            $('#teams_edit_or_create_team_error').css({ "display": "block" });
            $('#teams_edit_or_create_team_error').html("Error updating user");
            console.error(res);
        }).done(function (res) {
            console.log(res);
            $('#modal_edit_or_create_team').modal('hide');
            showSuccessNotification('Team updated successfully!');
            renderTeamsPage();
        });
    }
}

function renderTeamsPage() {
    $('.spa-web-page').css({"display": ""})
    $('#teams_page').css({"display": "block"})
    $('#teams_page_error').css({"display": "none"});
    $('#teams_page_error').html("");
    if (window.location.pathname != "/teams/") {
        window.location.href = "/teams/";
    }
    window.ctf01d_tp_api.teams_list().fail(function(res) {
        $('#teams_page_error').css({
            "display": "block"
        });
        $('#teams_page_error').html("Error loading services");
        console.error("teams_list", res);
    }).done(function(res) {
        var teamsHtml = ""
        for (var i in res) {
            var team_info = res[i];
            console.log("team_info", team_info);
            teamsHtml += '<div class="card services-card" style="width: 18rem;">';
            teamsHtml += '  <img class="card-img-top" src="' + team_info.avatar_url + '" alt="Image of service">';
            teamsHtml += '  <div class="card-body">';
            teamsHtml += '    <h5 class="card-title"><small>team#' + team_info.id + '</small><br>' + escapeHtml(team_info.name) + '</h5>'; // TODO uuid
            teamsHtml += '    <p class="card-text">' + escapeHtml(team_info.description) + '</p>';
            // teamsHtml += '    <p class="card-text"> by ' + escapeHtml(team_info.author) + '</p>';
            // TODO
            // teamsHtml += '    <small>' + getHumanTimeHasPassed(new Date(game_info.end_time)) + '</small>';
            teamsHtml += '  </div>';
            teamsHtml += '  <ul class="list-group list-group-flush">';
            teamsHtml += '    <li class="list-group-item">' + escapeHtml(team_info.university) + '</li>';
            // teamsHtml += '    <li class="list-group-item">Dapibus ac facilisis in</li>';
            // teamsHtml += '    <li class="list-group-item">Vestibulum at eros</li>';
            teamsHtml += '  </ul>';
            teamsHtml += '  <div class="card-body">';
            teamsHtml += '    <button class="btn btn-primary" onclick="showUpdateTeam(\'' + team_info.id + '\');">Update</button>';
            teamsHtml += '    <button class="btn btn-danger" onclick="deleteTeam(\'' + team_info.id + '\');">Delete</button>';
            teamsHtml += '  </div>';
            teamsHtml += '</div>';

            // teamsHtml += '  <div id="game_teams_' + game_info.id + '"> ' + new Date(game_info.end_time) + '</div>';
            // teamsHtml += '</div>';
            // updateGameTeams('service_teams_' + game_info.id, game_info.id)
        }
        $('#teams_page_list').html(teamsHtml);
    })
}

function renderPage(pathname) {
    console.log("pathname", pathname)
    if (pathname == "/") {
        $('.spa-web-page').css({
            "display": ""
        })
        $('#root_page').css({"display": "block"})
    } else if (isGamesPage(pathname)) {
        renderGamesPage();
    } else if (isServicesPage(pathname)) {
        renderServicesPage();
    } else if (isUsersPage(pathname)) {
        renderUsersPage();
    } else if (isTeamsPage(pathname)) {
        renderTeamsPage();
    } else {
        $('.spa-web-page').css({
            "display": ""
        })
        $('#unknown_page').css({"display": "block"})
    }
}

function showMyTeams(userId) {
    window.ctf01d_tp_api.user_profile(userId).fail(function (res) {
        $('#my_teams_content').html('<div class="alert alert-danger">Fail to fetch profile</div>');
        $('#modal_my_teams').modal('show');
        console.error("profile", res);
    }).done(function (data) {
        var teamHistoryHtml = '<div class="team-history">';
        for (var i in data.team_history) {
            var team = data.team_history[i];
            teamHistoryHtml += '<div class="team">';
            teamHistoryHtml += '<div class="team-name">[' + team.name + '] ' + team.role + '</div>';
            teamHistoryHtml += '<div class="team-dates">Joined at: ' + new Date(team.join).toLocaleString() +
                (team.left ? ', Left at: ' + new Date(team.left).toLocaleString() : ', ... ') + '</div>';
            teamHistoryHtml += '</div>';
        }
        teamHistoryHtml += '</div>';
        var currentTeamHtml = '<div class="current-team"><strong>Current Team: </strong>[' + data.team_name + '] ' + data.team_role + '</div>';
        $('#my_teams_content').html(currentTeamHtml + teamHistoryHtml);
        $('#modal_my_teams').modal('show');
    });
}

$(document).ready(function() {
    console.log("Ready");
    translateHtml();

    renderPage(window.location.pathname);

    window.ctf01d_tp_api.auth_session().fail(function(res) {
        console.error(res);
        $('#btn_sign_in').css({"display": "inline-block"});
        $('#btn_sign_out').css({"display": "none"});
        $('#btn_profile').css({"display": "none"});
    }).done(function(res) {
        $('#btn_sign_in').css({"display": "none"});
        $('#btn_sign_out').css({"display": "inline-block"});
        $('#btn_profile').css({"display": "inline-block"});
        $('#btn_profile').html(res.name + " (" + res.role + ")");
    })
})
