using UnityEngine;
using UnityEngine.UI;
using System.Collections;

public class Settings : MonoBehaviour {

	public GameObject loadingImage;
    public GameObject menuPanel;

    public void Start() {
        menuPanel.SetActive(false); // Hide settings panel on start.
    }

	public void LoadScene(string level)
	{
		UnityEngine.SceneManagement.SceneManager.LoadScene(level);
	}

    public void TogglePanel() 
    {
        // Toggle active state!
        menuPanel.SetActive(!menuPanel.activeSelf);
    }

    public void SetName() {
        GameObject textbox = GameObject.Find("usernameText");
        Text t = textbox.GetComponent<Text>();
        if (t.text == null || t.text == "") {
            // Random name?
            t.text = "Some Crazy Random Name";
        }
        PlayerPrefs.SetString("name", t.text);
    }

    public void SetIP() {
        GameObject textbox = GameObject.Find("iptext");
        Text t = textbox.GetComponent<Text>();
        string ip = t.text;
        if (t.text == null || t.text == "") {
            ip = "127.0.0.1";
        }
        Debug.Log("Setting IP to: " + ip);
        PlayerPrefs.SetString("ip", ip);
    }

    public void SaveSettings() {
        this.SetIP();
        this.TogglePanel();
    }
}
