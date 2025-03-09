import requests
from bs4 import BeautifulSoup
from string import ascii_lowercase
from urllib.parse import urljoin

session = requests.Session()
session.headers.update({
    "User-Agent": "Personal Project"
})

def generate_page(url):
    # Determine the pagination query parameter based on the site.
    if 'goodreads' in url:
        base_pagination = '?page='
    elif 'azquotes' in url:
        base_pagination = '?p='
    else:
        raise NotImplementedError("Only goodreads and azquotes supported")

    base_url = urljoin(url, base_pagination)
    i = 1
    while True:
        # Construct the full URL for page i.
        yield urljoin(base_url, str(i))
        i += 1

def scrape_topics_az_quotes():
    url = 'https://www.azquotes.com/quotes/tags/{}/'
    quote_url = 'https://www.azquotes.com'


    topics = []
    for letter in ascii_lowercase:
        topic_url = url.format(letter)
        response = session.get(topic_url, timeout=10)
        soup = BeautifulSoup(response.text, 'html.parser')
        soup = soup.find('section', class_='authors-page')
        for topic in soup.find_all('li'):
            topic_of_quote = topic.find('a').get_text(strip=True)
            topic_url = topic.find('a').get('href')
            if topic_of_quote:
                topics.append((topic_of_quote, urljoin(quote_url, topic_url)))

    return topics

def scrape_topics_goodreads():
    url = 'https://www.goodreads.com/quotes'
    response = session.get(url, timeout=10)
    soup = BeautifulSoup(response.text, 'html.parser')

    topics = []
    for topic in soup.find_all('li', class_='greyText'):
        topic_of_quote = topic.get_text(strip=True).split(' ')[0]
        topic_url = topic.find('a').get('href')
        topic_url = urljoin(url, topic_url)
        topics.append((topic_of_quote, topic_url))

    return topics

def scrape_authors_az_quotes():
    url = 'https://www.azquotes.com/quotes/authors/{}/'
    quote_url = 'https://www.azquotes.com'

    authors = []
    for letter in ascii_lowercase:
        author_letter_url = url.format(letter)
        response = session.get(author_letter_url + '1', timeout=10)
        soup = BeautifulSoup(response.text, 'html.parser')
        num_of_pages = soup.find('div', class_='pager').find('span').get_text().split(' ')[-1]
        for i in range(1, int(num_of_pages) + 1):
            response = session.get(author_letter_url + str(i), timeout=10)
            soup = BeautifulSoup(response.text, 'html.parser')
            table_body = soup.find('tbody')
            for row in table_body.find_all('tr'):
                columns = row.find_all('td')
                author_url = row.find('a').get('href')
                author_url = urljoin(quote_url, author_url)
                name, profession, birthday = [column.text.strip() for column in columns]
                authors.append((name, author_url, profession, birthday))

    return authors

if __name__ == '__main__':
    authors = scrape_authors_az_quotes()
    print(authors)