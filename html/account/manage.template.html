<link rel="stylesheet" href="/static/managestyle.css">

<article>
	<h2>Welcome {{ .User.Name }} !</h2>
	<h3> Your data </h3>
	<label>Age: {{ .User.Age }}</label>
	<label>Born: {{.User.Born}}</label>
	<h3>Links</h3>
	<table>
		<thead>
			<tr>
				<th>Short</th>
				<th>Long</th>
			</tr>
		</thead>
		
		{{ range .Links }} 
		<tr>
			<td><a href="{{ .Short }}">{{ .Short }}</a></td>
			<td><a href="{{ .Long }}">{{ .Long }}</a></td>
			<td><input class="delete-button" type="button" value="Delete" onclick="remove('{{ .Short }}')"></td>
		</tr>
		{{ end }}
	</table>

	<div id="message"></div>
	<form id="add_form">
		<div id="add-link-container">
			<label for="Short link">Short link</label>
			<input title="Short link" name="short" id="short-input" maxlength="6" type="text">
			<label for="Long link">Long link</label>
			<input title="Long link" placeholder="https://google.com" name="long" id="long-input" type="text">
		</div>
		<input type="button" value="Add" onclick="add()">
	</form>
</article>

<script>
	function add() {
		let req = new Request("/api/add", {
			method: "POST",
			body: new URLSearchParams(Object.fromEntries(new FormData(add_form))).toString(),
			headers: {
				"Content-Type" : "application/x-www-form-urlencoded",
				"Cookie": document.cookie
			}
		})

		fetch(req)
			.then(res => {
				if (res.status == 200) {
					location.reload()
				} else {
					message.innerHTML = "The entered short link was not valid"
				}
			})
	}

	function remove(shortUrl) {
		var url = new URL("/api/delete", location.origin)
		url.searchParams.append("short", shortUrl)

		console.log(url)

		let req = new Request(url, {
			method: "DELETE",
			headers: {
				"Content-Type" : "application/x-www-form-urlencoded",
				"Cookie": document.cookie
			}
		})

		fetch(req)
			.then(res => {
				if (res.status == 200) {
					location.reload()
				}
			})
	}
</script>